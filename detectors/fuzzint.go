package detectors

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"minievm/common"
	"os"
	"path"
	"strings"

	ui "github.com/gizak/termui"
	"github.com/google/gofuzz"
	"github.com/olekukonko/tablewriter"
)

/*
Try to fuzz input and storage, if we get a Overflow Event without revert, overflow detected.
*/
type FuzzInt struct {
	path, logpath, logfilename string
	contracts                  *ContractUtils
	fuzzer                     *fuzz.Fuzzer
	maincontract               *SimpleContract
	constantsLoc               map[string]common.Hash
	constantsName              []string
	enableUI                   bool
}

func GenRandomInSpecialDist() *big.Int {
	maxrange := int64(16)
	chooseRange, err := rand.Int(rand.Reader, big.NewInt(4))
	if err != nil {
		//error handling
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(maxrange))
	/*
		[0, 2^256-1] split into 3 parts
		[0, 2^10-1], [2^255-100, 2^255+99], [2^256-2^10, 2^256-1]
	*/
	// log.Printf("Mutating Storage %s\n", name)

	if chooseRange.Cmp(big.NewInt(0)) == 0 {
		// n = n
	} else if chooseRange.Cmp(big.NewInt(1)) == 0 {
		offset := new(big.Int)
		offset.Exp(big.NewInt(2), big.NewInt(255), nil).Sub(offset, big.NewInt(maxrange))
		n.Add(offset, n)
	} else if chooseRange.Cmp(big.NewInt(2)) == 0 {
		offset := new(big.Int)
		offset.Exp(big.NewInt(2), big.NewInt(255), nil)
		n.Add(offset, n)
	} else if chooseRange.Cmp(big.NewInt(3)) == 0 {
		offset := new(big.Int)
		offset.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(offset, big.NewInt(maxrange))
		n.Add(offset, n)
	}
	return n
}

func (fi *FuzzInt) FuzzStorage() {
	for _, loc := range fi.constantsLoc {
		n := GenRandomInSpecialDist()
		fi.contracts.SetStorage(loc, n)
	}
}

func NewContractFuzzer(solcpath, contractpath, logpath string, enableUI bool) *FuzzInt {
	fi := &FuzzInt{path: contractpath}
	fi.fuzzer = fuzz.New()
	fi.contracts = NewContract(solcpath, fi.path)
	fi.maincontract = &fi.contracts.MainContract
	fi.constantsLoc = fi.contracts.GetStorageLoc()
	fi.enableUI = enableUI
	patharray := strings.Split(fi.path, "/")
	fi.logfilename = "log_" + patharray[len(patharray)-1] + ".txt"
	fi.logpath = path.Join(logpath, fi.logfilename)
	for name := range fi.constantsLoc {
		fi.constantsName = append(fi.constantsName, name)
	}
	return fi
}

func (fi *FuzzInt) getConstantsTable() [][]string {
	rownum := len(fi.constantsName)
	table := make([][]string, rownum+1)
	for i := range table {
		table[i] = make([]string, 2)
	}
	table[0][0], table[0][1] = "Name", "Value"
	for i, name := range fi.constantsName {
		table[i+1][0], table[i+1][1] = name, fi.contracts.GetStorage(fi.constantsLoc[name]).String()
	}
	return table
}

func (fi *FuzzInt) GenTable(methodname, calldata string, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Type", "Name", "Value"})

	rownum := len(fi.constantsName)
	tableContent := make([][]string, rownum+1)
	for i := range tableContent {
		tableContent[i] = make([]string, 3)
	}
	for _, name := range fi.constantsName {
		table.Append([]string{"Storage", name, fi.contracts.GetStorage(fi.constantsLoc[name]).String()})
	}
	table.Append([]string{"Method", methodname, calldata})

	table.SetAutoMergeCells(true)
	table.Render() // Send output
}

const (
	StorageMutateNum = 100
	MaxRuns          = 100
)

func (fi *FuzzInt) CheckEvent() bool {
	overflowTopic := common.Hex2Bytes("FEE46111846A282E8199035721DD0334C2BC5C016AE4E72B924003431D6A8759")
	for _, logentry := range fi.contracts.state.Logs {
		if len(logentry.Topics) > 0 {
			logbytes := logentry.Topics[0].Bytes()
			if bytes.Equal(overflowTopic, logbytes) {
				return true
			}
		} else {
			return false
		}

	}
	return false
}

func (fi *FuzzInt) CheckOverflowStorage() bool {
	ops := []string{add, sub, mul, div}

	for _, op := range ops {
		res := fi.maincontract.evm.StateDB.GetState(fi.maincontract.Address, common.StringToHash(op))
		emptyHash := common.Hash{}
		if res != emptyHash {
			// log.Printf("Attack Vector: %02x\nUnpacked Input: %02x", packed, product)
			// log.Println(op[10:], "@PC =", res.Big())
			// log.Println()
			return true
		}
	}
	// ret, err = ofd.CallFunctionStub("stub_overflowdetection")
	// evm.StateDB.SetState(contract.Address(), common.StringToHash(key), common.BigToHash(big.NewInt(int64(*pc))))

	return false
}

func (fi *FuzzInt) FuzzContracts() {
	// log.Print(fi.logpath)
	f, err := os.Create(fi.logpath)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	if fi.enableUI {
		err = ui.Init()
		if err != nil {
			panic(err)
		}
		defer ui.Close()
	}

	if fi.enableUI {
		ui.Handle("/sys/kbd/q", func(ui.Event) {
			// press q to quit
			ui.StopLoop()
		})
	}

	p := ui.NewPar(":PRESS q TO QUIT")
	p.Height = 3
	p.Width = 85
	p.TextFgColor = ui.ColorWhite
	p.BorderLabel = "Simple Fuzzer"
	p.BorderFg = ui.ColorCyan

	table := ui.NewTable()
	table.Rows = fi.getConstantsTable()
	table.FgColor = ui.ColorWhite
	table.BgColor = ui.ColorDefault
	table.Width = 85
	table.Y = 4
	table.X = 0
	table.Border = true

	g := ui.NewGauge()
	g.Width = 50
	g.Height = 3
	g.BarColor = ui.ColorRed
	g.BorderFg = ui.ColorWhite
	g.BorderLabelFg = ui.ColorCyan

	gTotal := ui.NewGauge()
	gTotal.Width = 50
	gTotal.Height = 3
	gTotal.BorderLabel = "Total"
	gTotal.BarColor = ui.ColorYellow
	gTotal.BorderFg = ui.ColorWhite
	gTotal.BorderLabelFg = ui.ColorCyan
	evmLable := ui.NewPar("Waiting for fugitives")
	evmLable.BorderLabel = "EVM Result"
	evmLable.Height = 6
	evmLable.Width = 50
	evmLable.TextFgColor = ui.ColorWhite

	calldataLable := ui.NewPar("")
	calldataLable.X = g.Width + 1
	calldataLable.BorderLabel = "CallData"
	calldataLable.Height = 14
	calldataLable.Width = table.Width - g.Width - 1
	calldataLable.TextFgColor = ui.ColorWhite
	// evmLable.Text

	// log.Print(fi.maincontract.Name)
	for _, method := range fi.maincontract.ABI.Methods {
		if method.Const {
			continue
		}
		attackVectorCnt := 0
		g.BorderLabel = method.Name
	methodFuzz:
		for i := 0; i < StorageMutateNum; i++ {
			fi.FuzzStorage()
			table.Rows = fi.getConstantsTable()
			table.Height = len(table.Rows)*2 + 1
			g.Y = table.Y + table.Height + 1
			calldataLable.Y = g.Y
			gTotal.Y = g.Y + g.Height + 1
			evmLable.Y = gTotal.Y + gTotal.Height + 1
			if fi.enableUI {
				ui.Render(evmLable)
			}
			for j := 0; j < MaxRuns; j++ {
				g.Percent = j * 100 / MaxRuns
				gTotal.Percent = (i*MaxRuns + j) * 100 / (MaxRuns * StorageMutateNum)
				if fi.enableUI {
					ui.Render(g)
				}
				calldata, _ := method.Fuzz(fi.fuzzer)

				fi.contracts.BackupStates()
				_, err := fi.maincontract.Call(fi.contracts.ContractCreater, calldata)
				// PrintMemUsage()
				// log.Printf("Call Func: %s with %02x\n", method.Name, calldata)
				calldataLable.Text = common.ToHex(calldata)
				// eventExist := fi.CheckEvent() // require src transformer
				eventExist := false
				overflowStateExist := fi.CheckOverflowStorage()
				fi.contracts.RestoreStates()

				if err == nil {
					if eventExist {
						evmLable.Text = "Non-revert Detected\n" + calldataLable.Text + "\n"
						// evmLable.Text += fi.contracts.GetStorage(fi.constantsLoc["sellPrice"]).String()
						evmLable.Text += "\nEvent: " + fmt.Sprintf("%v", eventExist)
						if fi.enableUI {
							ui.Render(evmLable, calldataLable)
						}
						fi.GenTable(method.Sig(), common.ToHex(calldata), w)
						attackVectorCnt++
						// result:= strings.Sprintf("Current state: %s\nInput: %s\n",
					} else if overflowStateExist {
						evmLable.Text = "Non-revert Detected\n" + calldataLable.Text + "\n"
						// evmLable.Text += fi.contracts.GetStorage(fi.constantsLoc["sellPrice"]).String()
						evmLable.Text += "\nOverflow State: " + fmt.Sprintf("%v", overflowStateExist)
						if fi.enableUI {
							ui.Render(evmLable, calldataLable)
						}
						fi.GenTable(method.Sig(), common.ToHex(calldata), w)
						attackVectorCnt++
						// result:= strings.Sprintf("Current state: %s\nInput: %s\n",
					}
				}
				// method.Fuzz(fi.fuzzer)

				// log.Printf("name :%s, %02x\n", method.Sig(), calldata)
				if j == 0 {
					if fi.enableUI {
						ui.Render(p, table, gTotal, calldataLable)
					}
				}

				if attackVectorCnt > 5 {
					w.Flush()
					break methodFuzz
				}
			}
		}
	}
	if fi.enableUI {
		ui.Loop()
	}
}

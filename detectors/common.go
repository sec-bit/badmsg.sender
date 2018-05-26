package detectors

import (
	"math/big"
	"minievm/accounts/abi"
	"minievm/common"
	"reflect"
	"strconv"

	cartesian "github.com/schwarmco/go-cartesian-product"
)

const (
	keyPrefix = "Nyanpass. Overflow detected: "
	add       = keyPrefix + "Add"
	sub       = keyPrefix + "Sub"
	mul       = keyPrefix + "Mul"
	div       = keyPrefix + "Div"
)

func GenerateIntBySize(size int) (res []*big.Int) {
	value := new(big.Int)
	// 2**0----2**8
	for i := 0; i < 9; i++ {
		res = append(res, value.Exp(big.NewInt(2), big.NewInt(int64(i)), nil))
	}

	// last 8 2**size numbers
	for i := size - 8; i < size+1; i++ {
		res = append(res, value.Exp(big.NewInt(2), big.NewInt(int64(i)), nil))
	}
	// Temporary disable random

	intMax := new(big.Int)
	intMax.Exp(big.NewInt(2), big.NewInt(int64(size)), nil)

	// first 8 values
	for i := 1; i < 9; i++ {
		res = append(res, big.NewInt(int64(i)))
	}

	// last 8 values
	for i := 0; i < 9; i++ {
		tmp := new(big.Int)
		res = append(res, tmp.Sub(intMax, big.NewInt(int64(i))))
	}

	// // random big
	// 	res = append(res, value.Exp(big.NewInt(2), big.NewInt(int64(i)), nil))

	return
}

func GenerateAddress() (res []common.Address) {
	for i := 0; i < 10; i++ {
		res = append(res, common.BytesToAddress([]byte("testaddress"+strconv.Itoa(i))))
	}
	return
}

func GenerateAddresses() (res [][]common.Address) {
	var resTemp []common.Address
	for i := 0; i < 10; i++ {
		resTemp = append(resTemp, common.BytesToAddress([]byte("testaddress"+strconv.Itoa(i))))
	}
	for i := 0; i < 10; i++ {
		res = append(res, resTemp[:i])
	}
	return
}

// Type enumerator
const (
	IntTy byte = iota
	UintTy
	BoolTy
	StringTy
	SliceTy
	ArrayTy
	AddressTy
	FixedBytesTy
	BytesTy
	HashTy
	FixedPointTy
	FunctionTy
)

func GenerateInputsByABI(abi abi.ABI, function string) (c chan []interface{}, types []reflect.Type, total int) {
	total = 1
	var d [][]interface{}
	// IntTy byte = iota
	// UintTy
	// BoolTy
	// StringTy
	// SliceTy
	// ArrayTy
	// AddressTy
	// FixedBytesTy
	// BytesTy
	// HashTy
	// FixedPointTy
	// FunctionTy
	for _, x := range abi.Methods[function].Inputs {
		// log.Printf("%s, %d\n", x.Type, x.Type.Size)
		types = append(types, x.Type.Type)
		switch x.Type.T {
		case SliceTy:
			{
				switch x.Type.Elem.T {
				case AddressTy:
					{
						addresses := GenerateAddresses()
						s := make([]interface{}, len(addresses))
						for i, v := range addresses {
							s[i] = v
						}
						d = append(d, s)
						total *= len(s)
					}
				}
			}
		case UintTy:
			{
				numbers := GenerateIntBySize(x.Type.Size)
				s := make([]interface{}, len(numbers))
				for i, v := range numbers {
					s[i] = v
				}
				d = append(d, s)
				total *= len(s)
			}
		case AddressTy:
			{
				// addresses := []common.Address{common.BytesToAddress([]byte("Attacker001")),
				// 	common.BytesToAddress([]byte("Attacker002")),
				// 	common.BytesToAddress([]byte("Attacker003"))}
				// s := make([]interface{}, len(addresses))
				// for i, v := range addresses {
				// 	s[i] = v
				// }
				// d = append(d, s)
				// total *= len(s)
				addresses := GenerateAddress()
				s := make([]interface{}, len(addresses))
				for i, v := range addresses {
					s[i] = v
				}
				d = append(d, s)
				total *= len(s)
			}
		}
	}
	c = cartesian.Iter(d...)
	return

}

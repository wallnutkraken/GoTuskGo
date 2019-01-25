package gomarkov

import (
	"strconv"
	"testing"
)

// Shift array left with Copy
func ShiftLeft1(p []string) {
	x := p[0]
	copy(p, p[1:])
	p[len(p)-1] = x
}

// Shift array right with Copy
func ShiftRight1(p []string) {
	x := p[len(p)-1]
	copy(p, p[:len(p)-1])
	p[0] = x
}

// Shift array left with cycle for
func ShiftLeft2(p []string) {
	x := p[0]
	for i := 0; i < len(p)-1; i++ {
		p[i] = p[i+1]
	}
	p[len(p)-1] = x
}

// Shift array right with cycle for
func ShiftRight2(p []string) {
	x := p[len(p)-1]
	for i := len(p) - 1; i > 0; i-- {
		p[i] = p[i-1]
	}
	p[0] = x
}

func Benchmark_Check_performance_of_array_shift_left_with_Cycle_For(t *testing.B) {
	//fmt.Printf("Shift Left (For): %d\n", t.N)
	p := make([]string, t.N)

	for i := 0; i < len(p); i++ {
		p[i] = strconv.Itoa(i)
	}

	ShiftLeft2(p)
}

func Benchmark_Check_performance_of_array_shift_right_with_Cycle_For(t *testing.B) {
	//fmt.Printf("Shift Right (For): %d\n", t.N)
	p := make([]string, t.N)

	for i := 0; i < len(p); i++ {
		p[i] = strconv.Itoa(i)
	}

	ShiftRight2(p)
}

func Benchmark_Check_performance_of_array_shift_left_with_copy(t *testing.B) {

	//fmt.Printf("Shift Left (Copy): %d\n", t.N)
	p := make([]string, t.N)

	for i := 0; i < len(p); i++ {
		p[i] = strconv.Itoa(i)
	}

	ShiftLeft1(p)
}

func Benchmark_Check_performance_of_array_shift_right_with_copy(t *testing.B) {

	//fmt.Printf("Shift Right (Copy): %d\n", t.N)
	p := make([]string, t.N)

	for i := 0; i < len(p); i++ {
		p[i] = strconv.Itoa(i)
	}

	ShiftRight1(p)
}

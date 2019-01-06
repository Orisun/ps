// Date         : 2018/12/9
// Author       : zhangchaoyang
// Description  : 
package test

import (
	"testing"
	"ParameterServer/util/distribution"
	"fmt"
)

func TestConstant(t *testing.T) {
	var c float64 = 4.7
	var inst distribution.Distribution = distribution.GetConstantInstance(c)
	if inst.DrawOnePoint() != c {
		t.Fail()
	}
}

func TestUniform(t *testing.T) {
	var ceil float64 = 2.0
	var floor float64 = 0.0
	var inst distribution.Distribution = distribution.GetUniformInstance(floor, ceil)
	for i := 0; i < 10; i++ {
		fmt.Println(inst.DrawOnePoint())
	}
}

func TestGaussian(t *testing.T) {
	var mu float64 = 0.0
	var sigma float64 = 0.05
	var inst distribution.Distribution = distribution.GetGaussianInstance(mu, sigma)
	for i := 0; i < 10; i++ {
		fmt.Println(inst.DrawOnePoint())
	}
}

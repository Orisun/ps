// Date         : 2018/12/9
// Author       : zhangchaoyang
// Description  : 
package distribution

import (
	"math/rand"
	"math"
)

type Uniform struct {
	floor    float64
	ceil     float64
	interval float64
}

func GetUniformInstance(floor float64, ceil float64) *Uniform {
	if ceil <= floor {
		panic("invalid floor and ceil for uniform distribution")
		return nil
	}
	inst := new(Uniform)
	inst.ceil = ceil
	inst.floor = floor
	inst.interval = ceil - floor
	return inst
}

//DrawOnePoint 从均匀分布中随机抽取一个点
func (self *Uniform) DrawOnePoint() float64 {
	return rand.Float64()*self.interval + self.floor
}

//GetExpection 期望
func (self *Uniform) GetExpection() float64 {
	return (self.ceil + self.floor) / 2.0
}

//GetVariance 方差
func (self *Uniform) GetVariance() float64 {
	return float64(math.Pow(float64(self.ceil-self.floor), 2.0) / 12.0)
}

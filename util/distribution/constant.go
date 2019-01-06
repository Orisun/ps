// Date         : 2018/12/9
// Author       : zhangchaoyang
// Description  : 
package distribution

type Constant struct {
	v float64
}

func GetConstantInstance(v float64) *Constant {
	inst := new(Constant)
	inst.v = v
	return inst
}

//DrawOnePoint 从分布中随机抽取一个点
func (self *Constant) DrawOnePoint() float64 {
	return self.v
}

//GetExpection 期望
func (self *Constant) GetExpection() float64 {
	return self.v
}

//GetVariance 方差
func (self *Constant) GetVariance() float64 {
	return 0.0
}

// Date         : 2018/12/9
// Author       : zhangchaoyang
// Description  : 
package distribution

type Distribution interface {
	//从分布中随机抽取一个点
	DrawOnePoint() float64
	//分布的期望
	GetExpection() float64
	//分布的方差
	GetVariance() float64
}
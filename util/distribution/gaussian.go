// Date         : 2018/12/9
// Author       : zhangchaoyang
// Description  : 
package distribution

import (
	"math/rand"
	"math"
)

type Gaussian struct {
	mu    float64 //均值
	sigma float64 //标准差
	xmin  float64
	xmax  float64
	xspan float64
	ymin  float64
	ymax  float64
	yspan float64
}

func GetGaussianInstance(mu float64, sigma float64) *Gaussian {
	if sigma <= 0 {
		panic("not positive sigma for gaussian distribution")
		return nil
	}
	inst := new(Gaussian)
	inst.mu = mu
	inst.sigma = sigma
	// 理论上x的取值范围是[-inf,inf]，但实际上绝大部分的x都位于[-4*sigma,4*sigma]之间
	inst.xmin = -4 * sigma
	inst.xmax = 4 * sigma
	inst.xspan = inst.xmax - inst.xmin
	inst.ymin = 0
	inst.ymax = float64(1.0 / math.Sqrt(2*math.Pi))
	inst.yspan = inst.ymax - inst.ymin
	return inst
}

//pdf 概率密度函数
func (self *Gaussian) pdf(x float64) float64 {
	return float64(1.0 / (math.Sqrt(2*math.Pi) * float64(self.sigma)) * math.Exp(-math.Pow(float64(x-self.mu),
		2.0) / (2 * float64(self.sigma*self.sigma))))
}

//DrawOnePoint Acceptance-Rejection Method抽样法算法步骤如下:
//1. 设概率密度函数为f(x)，f(x)的定义域为[x_min,x_max]，值域为[y_min,y_max]
//2. 独立生成2个服从均匀分布的随机变量，X~Uni(x_min,x_max)，Y~Uni(y_min,y_max)
//3. 如果Y<=f(X)，则返回X；否则回到第1步
func (self *Gaussian) DrawOnePoint() float64 {
	for {
		x := self.xmin + self.xspan*rand.Float64()
		y := self.ymin + self.yspan*rand.Float64()
		if y < self.pdf(x) {
			return x
		}
	}
}

//GetExpection 期望
func (self *Gaussian) GetExpection() float64 {
	return self.mu
}

//GetVariance 方差
func (self *Gaussian) GetVariance() float64 {
	return self.sigma * self.sigma
}

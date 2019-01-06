// Date         : 2018/12/16
// Author       : zhangchaoyang
// Description  : 
package main

import (
	"math"
	"time"
	"ParameterServer/util"
)

func fn(x []float64, w []float64) float64 {
	var z float64 = 0.0
	if len(x) != len(w) {
		panic("length not match")
	}
	for i := 0; i < len(x); i++ {
		z += x[i] * w[i]
	}
	//var y_hat float64 = 1.0 / (1.0 + math.Pow(math.E, -z))
	var y_hat float64 = (1.0 + math.Tanh(z/2.0)) / 2.0
	return y_hat
}

func crossEntropy(y float64, y_hat float64) float64 {
	l := -y*math.Log(y_hat+NEAR_0) - (1-y)*math.Log(1-y_hat+NEAR_0)
	return l
}

func batchGradient(sampleBatch []*Sample, w []float64) []float64 {
	gSum := make([]float64, len(w))
	for _, sample := range sampleBatch {
		y_hat := fn(sample.X, w)
		sub := y_hat - sample.Y
		//util.Log.Debugf("y=%f, y_hat=%f", sample.Y, y_hat)
		//只需要计算自己负责的区间内的导数
		for i := 0; i < len(w); i++ {
			v := sample.X[i]
			gSum[i] += sub * v //各维度上梯度求和
		}
	}
	
	g := make([]float64, len(w))
	for i, v := range gSum {
		g[i] = v / float64(len(sampleBatch)) //各维度上梯度求平均值
	}
	return g
}

func trainFTRL(ParameterTotal int, corpusFile string, epoch int, batch int, alpha float64, beta float64, l1 float64,
	l2 float64) []float64 {
	begin := time.Now()
	z := make([]float64, ParameterTotal)
	n := make([]float64, ParameterTotal)
	w := make([]float64, ParameterTotal)
	
	for ep := 0; ep < epoch; ep++ {
		util.Log.Infof("epoch=%d", ep)
		CorpusGenerator := new(CorpusGenerator)
		CorpusGenerator.Init(1, 0, corpusFile, ParameterTotal)
		sampleBatch := []*Sample{}
		for {
			sample := CorpusGenerator.PeekOneSample()
			if sample == nil {
				CorpusGenerator.Close()
				break
			}
			sampleBatch = append(sampleBatch, sample)
			if len(sampleBatch) >= batch {
				for i := 0; i < ParameterTotal; i++ {
					if math.Abs(z[i]) < l1 {
						w[i] = 0.0
					} else {
						l := l1
						if math.Signbit(z[i]) {
							l = -l1
						}
						w[i] = -(z[i] - l) / (l2 + (beta+math.Sqrt(n[i]))/alpha)
					}
				}
				g := batchGradient(sampleBatch, w)
				for i := 0; i < len(g); i++ {
					sigma := (math.Sqrt(n[i]+g[i]*g[i]) - math.Sqrt(
						n[i])) / alpha
					z[i] += g[i] - sigma*w[i]
					n[i] += g[i] * g[i]
				}
				sampleBatch = []*Sample{}
			}
		}
	}
	util.Log.Infof("train lr with ftrl finished, use %f seconds", time.Since(begin).Seconds())
	return w
}

func predictByFTRL(w []float64, corpusFile string) {
	begin := time.Now()
	ParameterTotal := len(w)
	CorpusGenerator := new(CorpusGenerator)
	CorpusGenerator.Init(10, 0, corpusFile, ParameterTotal)
	loss := 0.0
	population := 0
	for {
		sample := CorpusGenerator.PeekOneSample()
		if sample == nil {
			CorpusGenerator.Close()
			break
		}
		y_hat := fn(sample.X, w)
		loss += crossEntropy(sample.Y, y_hat)
		population++
	}
	util.Log.Infof("predit %d samples, mean loss %f, use %f seconds", population, loss/float64(population),
		time.Since(begin).Seconds())
}
func main3() {
	w := trainFTRL(10000, "../data/binary_class.csv", 20, 64, 0.1, 1.0, 1.0, 1.0)
	predictByFTRL(w, "../data/binary_class.csv")
	util.Log.Flush()
}

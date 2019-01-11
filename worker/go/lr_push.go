// Date         : 2018/12/8
// Author       : zhangchaoyang
// Description  : 用Pull-Push模式，这不适合于梯度下降类的优化方法
package main

import (
	psclient "ParameterServer/client/go"
	communicate "ParameterServer/communicate/go"
	"ParameterServer/util"
	"ParameterServer/util/distribution"
	"flag"
	"fmt"
	"math"
	"syscall"
	"time"
)

const (
	NEAR_0 = 1e-10
)

type LR struct {
	psClient *psclient.PsClient
	KeyRange communicate.Range
}

func (self *LR) Init(manager string, port int, ParameterTotal int32, KeyRange communicate.Range,
	initParamFunc distribution.Distribution, valuesEachKey int) {
	self.psClient = new(psclient.PsClient)
	self.psClient.Init(manager, port, ParameterTotal, KeyRange)
	self.KeyRange = KeyRange
	//初始化参数，从PS上取
	msgId := self.psClient.Pull()
	pullOk := self.psClient.WaitPull(msgId, 5*time.Second)
	w := self.psClient.GetAllParameter()
	//如果没有从PS是取下参数，则使用initParamFunc初始化参数
	if !pullOk || w[KeyRange.Begin] == nil || len(w[KeyRange.Begin].Values) == 0 {
		if initParamFunc == nil {
			util.Log.Errorf("could not init parameter from ps, and init function is nil")
			util.Log.Flush()
			syscall.Exit(1)
		}
		initValues := []*communicate.Value{}
		for i := 0; i < int(ParameterTotal); i++ {
			l := make([]float64, valuesEachKey)
			for i := 0; i < valuesEachKey; i++ {
				l[i] = initParamFunc.DrawOnePoint()
			}
			value := communicate.Value{Values: l}
			initValues = append(initValues, &value)
		}
		self.psClient.UpdateLocalParameter(initValues)
		msgId := self.psClient.Push() //把初始化好的参数push到PS集群上
		self.psClient.WaitPush(msgId, 5*time.Second)
		util.Log.Infof("init parameter by init function")
	} else {
		util.Log.Infof("init parameter from ps")
	}
}

//fn LR决策函数
func (self *LR) fn(x []float64, w []float64) float64 {
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

//loss 交叉熵损失
func (self *LR) loss(y float64, y_hat float64) float64 {
	l := -y*math.Log(y_hat+NEAR_0) - (1-y)*math.Log(1-y_hat+NEAR_0)
	return l
}

//gradient 交叉熵损失函数对w的一阶导数
func (self *LR) gradient(y float64, y_hat float64, x []float64) []float64 {
	g := make([]float64, self.KeyRange.End-self.KeyRange.Begin)
	sub := y_hat - y
	//只需要计算自己负责的区间内的导数
	for i := self.KeyRange.Begin; i < self.KeyRange.End; i++ {
		v := x[i]
		g[i-self.KeyRange.Begin] = sub * v
	}
	return g
}

//batchGradient 基于小批量求梯度
func (self *LR) batchGradient(sampleBatch []*Sample, w []float64) []float64 {
	gSum := make([]float64, self.KeyRange.End-self.KeyRange.Begin)
	for _, sample := range sampleBatch {
		y_hat := self.fn(sample.X, w)
		sub := y_hat - sample.Y
		//util.Log.Debugf("y=%f, y_hat=%f", sample.Y, y_hat)
		//只需要计算自己负责的区间内的导数
		for i := self.KeyRange.Begin; i < self.KeyRange.End; i++ {
			v := sample.X[i]
			gSum[i-self.KeyRange.Begin] += sub * v //各维度上梯度求和
		}
	}

	g := make([]float64, self.KeyRange.End-self.KeyRange.Begin)
	for i, v := range gSum {
		g[i] = v / float64(len(sampleBatch)) //各维度上梯度求平均值
	}
	return g
}

func TrainLrWithGD(corpusFile string, splitNum int, splitIndex int, epoch int, batch int, eta float64,
	manager string, port int, ParameterTotal int32, KeyRange communicate.Range,
	initParamFunc distribution.Distribution, xDim int, verbose bool) *LR {
	begin := time.Now()
	lr := new(LR)
	lr.Init(manager, port, ParameterTotal, KeyRange, initParamFunc, 1)
	defer lr.psClient.ShutDown()
	iter := 1
	var WaitedPullMsgId int64 = 0
	for ep := 0; ep < epoch; ep++ {
		util.Log.Debugf("epoch=%d", ep)
		CorpusGenerator := new(CorpusGenerator)
		CorpusGenerator.Init(int32(splitNum), int32(splitIndex), corpusFile, xDim)
		sampleBatch := []*Sample{}
		for {
			sample := CorpusGenerator.PeekOneSample()
			if sample == nil {
				CorpusGenerator.Close()
				break
			}
			sampleBatch = append(sampleBatch, sample)
			if len(sampleBatch) >= batch {
				if WaitedPullMsgId > 0 {
					if !lr.psClient.WaitPull(WaitedPullMsgId, 1*time.Second) { //等5次迭代之前的Pull命令完成
						util.Log.Errorf("wait pull timeout")
					}
				}
				msgId := lr.psClient.Pull()
				if iter%5 == 0 {
					WaitedPullMsgId = msgId
				}
				iter++
				w := make([]float64, ParameterTotal) //从PS上取下全部的w
				for i, Value := range lr.psClient.GetAllParameter() {
					if len(Value.Values) >= 1 {
						w[i] = Value.Values[0]
					} else {
						util.Log.Errorf("parameters of one key less than 1: %d", len(Value.Values))
					}
				}
				g := lr.batchGradient(sampleBatch, w) //只需要计算部分梯度，g的长度为KeyRange.End-KeyRange.Begin

				if verbose {
					loss := 0.0
					for _, s := range sampleBatch {
						y_hat := lr.fn(s.X, w)
						loss += lr.loss(s.Y, y_hat)
					}
					util.Log.Infof("iter=%d, batch mean loss %f", iter, loss/float64(len(sampleBatch)))
				}

				Values := make([]*communicate.Value, KeyRange.End-KeyRange.Begin)
				for i, v := range g {
					//梯度下降法的核心公式
					value := communicate.Value{Values: []float64{w[i+int(KeyRange.Begin)] - eta*v}}
					Values[i] = &value
				}
				lr.psClient.UpdateLocalRangedParameter(Values)
				lr.psClient.Push()
				sampleBatch = []*Sample{}
			}
		}
		if len(sampleBatch) >= 1 {
			if WaitedPullMsgId > 0 {
				if !lr.psClient.WaitPull(WaitedPullMsgId, 1*time.Second) { //等5次迭代之前的Pull命令完成
					util.Log.Errorf("wait pull timeoutt")
				}
			}
			msgId := lr.psClient.Pull()
			if iter%5 == 0 {
				WaitedPullMsgId = msgId
			}
			iter++
			w := make([]float64, ParameterTotal) //从PS上取下全部的w
			for i, Value := range lr.psClient.GetAllParameter() {
				if len(Value.Values) >= 1 {
					w[i] = Value.Values[0]
				} else {
					util.Log.Errorf("parameters of one key less than 1: %d", len(Value.Values))
				}
			}
			g := lr.batchGradient(sampleBatch, w) //只需要计算部分梯度，g的长度为KeyRange.End-KeyRange.Begin

			if verbose {
				loss := 0.0
				for _, s := range sampleBatch {
					y_hat := lr.fn(s.X, w)
					loss += lr.loss(s.Y, y_hat)
				}
				util.Log.Infof("iter=%d, batch mean loss %f", iter, loss/float64(len(sampleBatch)))
			}

			Values := make([]*communicate.Value, KeyRange.End-KeyRange.Begin)
			for i, v := range g {
				//梯度下降法的核心公式
				value := communicate.Value{Values: []float64{w[i+int(KeyRange.Begin)] - eta*v}}
				Values[i] = &value
			}
			lr.psClient.UpdateLocalRangedParameter(Values)
			lr.psClient.Push()
		}
	}
	util.Log.Infof("train lr with gd finished, use %f seconds", time.Since(begin).Seconds())
	return lr
}

func TrainLrWithFTRL(corpusFile string, splitNum int, splitIndex int, epoch int, batch int,
	manager string, port int, ParameterTotal int32, KeyRange communicate.Range,
	initParamFunc distribution.Distribution, xDim int, alpha float64, beta float64, l1 float64, l2 float64,
	verbose bool) *LR {
	begin := time.Now()
	lr := new(LR)
	lr.Init(manager, port, ParameterTotal, KeyRange, initParamFunc, 2)
	iter := 1
	var WaitedPullMsgId int64 = 0

	z := make([]float64, ParameterTotal)
	n := make([]float64, ParameterTotal)
	w := make([]float64, ParameterTotal)

	for ep := 0; ep < epoch; ep++ {
		util.Log.Debugf("epoch=%d", ep)
		CorpusGenerator := new(CorpusGenerator)
		CorpusGenerator.Init(int32(splitNum), int32(splitIndex), corpusFile, xDim)
		sampleBatch := []*Sample{}
		for {
			sample := CorpusGenerator.PeekOneSample()
			if sample == nil {
				CorpusGenerator.Close()
				break
			}
			sampleBatch = append(sampleBatch, sample)
			if len(sampleBatch) >= batch {
				if WaitedPullMsgId > 0 {
					if !lr.psClient.WaitPull(WaitedPullMsgId, 1*time.Second) { //等5次迭代之前的Pull命令完成
						util.Log.Errorf("wait pull timeoutt")
					}
				}
				msgId := lr.psClient.Pull()
				if iter%5 == 0 {
					WaitedPullMsgId = msgId
				}

				iter++
				for i, Value := range lr.psClient.GetAllParameter() {
					if len(Value.Values) >= 2 {
						z[i] = Value.Values[0]
						n[i] = Value.Values[1]
					} else {
						util.Log.Errorf("parameters of one key less than 2: %d", len(Value.Values))
					}
				}
				//FTRL的核心公式
				for i := 0; i < int(ParameterTotal); i++ {
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

				if verbose {
					loss := 0.0
					for _, s := range sampleBatch {
						y_hat := lr.fn(s.X, w)
						loss += lr.loss(s.Y, y_hat)
					}
					util.Log.Infof("iter=%d, batch mean loss %f", iter, loss/float64(len(sampleBatch)))
				}

				Values := make([]*communicate.Value, KeyRange.End-KeyRange.Begin)
				g := lr.batchGradient(sampleBatch, w) //只需要计算部分梯度，g的长度为KeyRange.End-KeyRange.Begin
				for i := 0; i < len(g); i++ {
					//只更新自己负责的区间段
					sigma := (math.Sqrt(n[i+int(KeyRange.Begin)]+g[i]*g[i]) - math.Sqrt(
						n[i+int(KeyRange.Begin)])) / alpha
					z[i+int(KeyRange.Begin)] += g[i] - sigma*w[i+int(KeyRange.Begin)]
					n[i+int(KeyRange.Begin)] += g[i] * g[i]

					value := communicate.Value{Values: []float64{z[i+int(KeyRange.Begin)], n[i+int(KeyRange.Begin)]}}
					Values[i] = &value
				}
				lr.psClient.UpdateLocalRangedParameter(Values)
				lr.psClient.Push()
				sampleBatch = []*Sample{}
			}
		}
		if len(sampleBatch) >= 1 {
			if WaitedPullMsgId > 0 {
				if !lr.psClient.WaitPull(WaitedPullMsgId, 1*time.Second) { //等5次迭代之前的Pull命令完成
					util.Log.Errorf("wait pull timeoutt")
				}
			}
			msgId := lr.psClient.Pull()
			if iter%5 == 0 {
				WaitedPullMsgId = msgId
			}

			iter++
			for i, Value := range lr.psClient.GetAllParameter() {
				if len(Value.Values) >= 2 {
					z[i] = Value.Values[0]
					n[i] = Value.Values[1]
				} else {
					util.Log.Errorf("parameters of one key less than 2: %d", len(Value.Values))
				}
			}
			//FTRL的核心公式
			for i := 0; i < int(ParameterTotal); i++ {
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

			if verbose {
				loss := 0.0
				for _, s := range sampleBatch {
					y_hat := lr.fn(s.X, w)
					loss += lr.loss(s.Y, y_hat)
				}
				util.Log.Infof("iter=%d, batch mean loss %f", iter, loss/float64(len(sampleBatch)))
			}

			Values := make([]*communicate.Value, KeyRange.End-KeyRange.Begin)
			g := lr.batchGradient(sampleBatch, w) //只需要计算部分梯度
			for i := 0; i < len(g); i++ {
				//只更新自己负责的区间段
				sigma := (math.Sqrt(n[i+int(KeyRange.Begin)]+g[i]*g[i]) - math.Sqrt(
					n[i+int(KeyRange.Begin)])) / alpha
				z[i+int(KeyRange.Begin)] += g[i] - sigma*w[i+int(KeyRange.Begin)]
				n[i+int(KeyRange.Begin)] += g[i] * g[i]

				value := communicate.Value{Values: []float64{z[i+int(KeyRange.Begin)], n[i+int(KeyRange.Begin)]}}
				Values[i] = &value
			}
			lr.psClient.UpdateLocalRangedParameter(Values)
			lr.psClient.Push()
		}
	}
	util.Log.Infof("train lr with ftrl finished, use %f seconds", time.Since(begin).Seconds())
	return lr
}

func PreditByLR(lr *LR, corpusFile string, splitNum int, splitIndex int,
	alpha float64, beta float64, l1 float64, l2 float64) {
	begin := time.Now()
	ParameterTotal := lr.psClient.ParameterTotal
	CorpusGenerator := new(CorpusGenerator)
	CorpusGenerator.Init(int32(splitNum), int32(splitIndex), corpusFile, int(ParameterTotal))
	var loss float64 = 0.0
	population := 0
	w := make([]float64, 0, ParameterTotal)
	z := make([]float64, 0, ParameterTotal)
	n := make([]float64, 0, ParameterTotal)
	for _, Value := range lr.psClient.GetAllParameter() {
		if len(Value.Values) == 1 {
			w = append(w, Value.Values[0])
		} else if len(Value.Values) == 2 {
			z = append(z, Value.Values[0])
			n = append(n, Value.Values[1])
		} else {
			panic("invalid values length")
		}
	}
	if len(w) == 0 {
		util.Log.Infof("model is trained by ftrl")
		//根据n和z计算得到w
		none_zero := 0
		for i := 0; i < len(z); i++ {
			if math.Abs(z[i]) < l1 {
				w = append(w, 0.0)
			} else {
				l := l1
				if math.Signbit(z[i]) {
					l = -l1
				}
				w = append(w, -(z[i] - l)/(l2+(beta+math.Sqrt(n[i]))/alpha))
				none_zero++
			}
		}
		util.Log.Infof("got %d w, none_zero %d", len(w), none_zero)
	} else {
		util.Log.Infof("model is trained by gd")
	}
	for {
		sample := CorpusGenerator.PeekOneSample()
		if sample == nil {
			CorpusGenerator.Close()
			break
		}
		y_hat := lr.fn(sample.X, w)
		loss += lr.loss(sample.Y, y_hat)
		population++
	}
	util.Log.Infof("predit %d samples, mean loss %f, use %f seconds", population, loss/float64(population),
		time.Since(begin).Seconds())
}

func main1() {
	port := flag.Int("port", 0, "work port")
	manager := flag.String("manager", "", "manager host")
	parameterCount := flag.Int("parameter_count", 0, "total parameter count")
	corpusFile := flag.String("corpus_file", "", "corpus file")
	corpusSplitNum := flag.Int("corpus_split_num", 1, "corpus split count")
	corpusSplitIndex := flag.Int("corpus_split_index", 0, "read which index of corpus splits")
	featureSplitNum := flag.Int("feature_split_num", 1, "feature split count")
	featreSplitIndex := flag.Int("feature_split_index", 0, "train which index of feature splits")
	epoch := flag.Int("epoch", 20, "train epoch")
	flag.Parse()

	fmt.Printf("work port %d\n", *port)
	fmt.Printf("manager host %s\n", *manager)
	fmt.Printf("total parameter count %d\n", *parameterCount)
	fmt.Printf("corpus file %s\n", *corpusFile)
	fmt.Printf("corpus split count %d\n", *corpusSplitNum)
	fmt.Printf("read which index of corpus splits %d\n", *corpusSplitIndex)
	fmt.Printf("feature split count %d\n", *featureSplitNum)
	fmt.Printf("train which index of feature splits %d\n", *featreSplitIndex)
	fmt.Printf("train epoch %d\n", *epoch)

	if *port <= 10000 {
		fmt.Printf("could not bind to port less than 10000")
		return
	}
	featureSplitSize := (*parameterCount) / (*featureSplitNum)
	keyBegin := (*featreSplitIndex) * featureSplitSize
	keyEnd := (*featreSplitIndex + 1) * featureSplitSize
	if (*featreSplitIndex)+1 == (*featureSplitNum) {
		keyEnd = (*parameterCount)
	}
	KeyRange := communicate.Range{Begin: int32(keyBegin), End: int32(keyEnd)}
	//initParamFunc := distribution.GetGaussianInstance(0.0, 0.05) //按正态分布初始化参数
	constParmFunc := distribution.GetConstantInstance(0.0) //用常数0初始化参数
	var lr *LR
	//lr = TrainLrWithGD(*corpusFile, *corpusSplitNum, *corpusSplitIndex, *epoch, 64, 1e-1,
	//	*manager, *port, int32(*parameterCount), KeyRange, initParamFunc, *parameterCount, false)
	lr = TrainLrWithFTRL(*corpusFile, *corpusSplitNum, *corpusSplitIndex, *epoch, 64,
		*manager, *port, int32(*parameterCount), KeyRange, constParmFunc, *parameterCount, 0.1, 1.0, 1.0, 1.0,
		false)                                             //FTRL中的z和n初始化为0，注意n一定不能是负数
	PreditByLR(lr, *corpusFile, 10, 0, 0.1, 1.0, 1.0, 1.0) //取1/10的样本做预测
	lr.psClient.ShutDown()
	util.Log.Flush()
}

//读1/3的样本
//go clean && go build -o lr && ./lr --port 40010 --manager jobflume --parameter_count 10000 --corpus_file ../data/binary_class.csv  --epoch 20 --feature_split_num 1 --feature_split_index 0 --corpus_split_num 3 --corpus_split_index 0

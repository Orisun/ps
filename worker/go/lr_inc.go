// Date         : 2018/12/23
// Author       : zhangchaoyang
// Description  : 用Pull-Inc模式
package main

import (
	communicate "ParameterServer/communicate/go"
	"ParameterServer/util"
	"ParameterServer/util/distribution"
	"flag"
	"fmt"
	"math"
	"time"
)

func TrainLrWithGD2(corpusFile string, splitNum int, splitIndex int, epoch int, batch int, eta float64,
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
					value := communicate.Value{Values: []float64{- eta * v}}
					Values[i] = &value
				}
				lr.psClient.Inc(Values)
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
				value := communicate.Value{Values: []float64{- eta * v}}
				Values[i] = &value
			}
			lr.psClient.Inc(Values)
		}
	}
	util.Log.Infof("train lr with gd finished, use %f seconds", time.Since(begin).Seconds())
	return lr
}

func TrainLrWithFTRL2(corpusFile string, splitNum int, splitIndex int, epoch int, batch int,
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
						//util.Log.Debugf("i=%d, z=%f, n=%f, w=%f", i, z[i], n[i], w[i])
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
					//util.Log.Debugf("n=%f, g=%f", n[i+int(KeyRange.Begin)], g[i])
					//只更新自己负责的区间段
					sigma := (math.Sqrt(n[i+int(KeyRange.Begin)]+g[i]*g[i]) - math.Sqrt(
						n[i+int(KeyRange.Begin)])) / alpha
					delta_z := g[i] - sigma*w[i+int(KeyRange.Begin)]
					//util.Log.Debugf("i=%d, sigma=%f, w=%f, delta_z=%f", i, sigma, w[i+int(KeyRange.Begin)], delta_z)
					delta_n := g[i] * g[i]
					
					value := communicate.Value{Values: []float64{delta_z, delta_n}}
					Values[i] = &value
				}
				lr.psClient.Inc(Values)
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
				delta_z := g[i] - sigma*w[i+int(KeyRange.Begin)]
				delta_n := g[i] * g[i]
				
				value := communicate.Value{Values: []float64{delta_z, delta_n}}
				Values[i] = &value
			}
			lr.psClient.Inc(Values)
		}
	}
	util.Log.Infof("train lr with ftrl finished, use %f seconds", time.Since(begin).Seconds())
	return lr
}

func main() {
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
	//lr = TrainLrWithGD2(*corpusFile, *corpusSplitNum, *corpusSplitIndex, *epoch, 64, 1e-1,
	//	*manager, *port, int32(*parameterCount), KeyRange, initParamFunc, *parameterCount, false)
	lr = TrainLrWithFTRL2(*corpusFile, *corpusSplitNum, *corpusSplitIndex, *epoch, 64,
		*manager, *port, int32(*parameterCount), KeyRange, constParmFunc, *parameterCount, 0.1, 1.0, 1.0, 1.0,
		false)                                              //FTRL中的z和n初始化为0，注意n一定不能是负数
	PreditByLR(lr, *corpusFile, 10, 0, 0.1, 1.0, 1.0, 1.0) //取1/10的样本做预测
	lr.psClient.ShutDown()
	util.Log.Flush()
}

//读1/3的样本，训练全部的参数
//go clean && go build -o lr && ./lr --port 40010 --manager jobflume --parameter_count 10000 --corpus_file ../data/binary_class.csv  --epoch 20 --feature_split_num 1 --feature_split_index 0 --corpus_split_num 3 --corpus_split_index 0

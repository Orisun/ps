// Date         : 2018/12/8
// Author       : zhangchaoyang
// Description  : 
package main

import (
	"os"
	"bufio"
	"strings"
	"strconv"
	"ParameterServer/util"
	"math/rand"
)

type Sample struct {
	X []float64
	Y float64
}

type CorpusGenerator struct {
	dataBuffer chan Sample
	splitNum   int32 //把训练数据分成splitNum份
	index      int32 //这里只读取第index份
	fin        *os.File
	readOver   chan int
	xDim       int
}

func (self *CorpusGenerator) Init(splitNum int32, index int32, corpusFile string, xDim int) {
	self.dataBuffer = make(chan Sample, 100)
	self.splitNum = splitNum
	self.index = index
	self.readOver = make(chan int, 1)
	self.xDim = xDim
	f, err := os.Open(corpusFile)
	if err != nil {
		panic(err)
	}
	self.fin = f
	
	go func() {
		self.readFile()
	}()
}

//PeekOneSample 取走一条样本，如果文件已读完则返回nil
func (self *CorpusGenerator) PeekOneSample() *Sample {
	select {
	case s := <-self.dataBuffer:
		return &s
	case <-self.readOver:
		return nil
	}
}

func (self *CorpusGenerator) readFile() {
	buf := bufio.NewReader(self.fin)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			util.Log.Errorf("read file end with err %s", err.Error())
			break
		}
		if rand.Int31n(self.splitNum) == self.index {
			if len(line) == 0 || line[0] == '#' {
				continue
			}
			arr := strings.Split(strings.TrimSpace(line), ",")
			if len(arr) == self.xDim+1 {
				x := make([]float64, self.xDim)
				for i := 0; i < self.xDim; i++ {
					v, _ := strconv.ParseFloat(arr[i], 64)
					x[i] = v
				}
				y, _ := strconv.ParseFloat(arr[len(arr)-1], 64)
				sample := Sample{
					X: x,
					Y: y,
				}
				self.dataBuffer <- sample //如果channel已满，这里会阻塞
			}
		}
	}
	self.readOver <- 1
}

func (self *CorpusGenerator) Close() {
	self.fin.Close()
	close(self.dataBuffer)
}

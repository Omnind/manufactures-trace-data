package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
)

var (
	ftime string
)

func main() {
	fmt.Println("Please enter the expected time: ")
	fmt.Scanln(&ftime)
	//time.Now().Format("2006-01-02")
	f, err := os.Open("test_out.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	csvFile := csv.NewReader(f)

	csvOut, err := os.Create("Distribution/" + ftime + "_Distribution.csv") //ftime
	if err != nil {
		panic(err)
	}
	defer csvOut.Close()

	ss, err := csvFile.Read()
	//m := map[int]string{}
	//for i := 16; i < len(ss); i++ {
	//	m[i] = ss[i]
	//}

	for i := 0; i < len(ss); i++ {
		csvOut.WriteString(ss[i])
		csvOut.WriteString(",")
	}
	csvOut.WriteString("\n")

	for {
		ss, err = csvFile.Read()
		if err != nil {
			break
		}
		nums := make([]int, 0, 10000)
		sum := 0
		weight := 0
		weightSum := 0
		for i, indexWeights := 16, 1; i < len(ss); i, indexWeights = i+1, indexWeights+1 {
			t, err := strconv.Atoi(ss[i])
			if err != nil {
				log.Println("Num Transformation failed")
				continue
			}
			nums = append(nums, t)
			sum += t
			weight += indexWeights * t
			weightSum += indexWeights
		}

		avg := fmt.Sprintf("%.3f", float64(weight)/float64(sum))

		zhe := []float64{0.1, 0.25, 0.5, 0.75, 0.9}
		th := []int{}
		for i := 0; i < len(zhe); i++ {
			th = append(th, int(zhe[i]*float64(sum)))
		}

		th_results := make([]int, 5)
		th_cha := make([]int, 5)

		for i := 0; i < len(th_cha); i++ {
			th_cha[i] = sum
		}

		max := 0
		min := 0
		maxIndex := 0
		minIndex := 0
		sumIndex := 0
		for i := 0; i < len(nums); i++ {

			if min == 0 {
				if nums[i] != 0 {
					min = nums[i]
					minIndex = i
				}
			}

			sumIndex += nums[i]
			for j := 0; j < len(th); j++ {
				if sumIndex > th[j] {
					th_results[j] = i
					th[j] = math.MaxInt
				}
			}
		}
		for i := len(nums) - 1; i >= 0; i-- {
			if max == 0 {
				if nums[i] != 0 {
					max = nums[i]
					maxIndex = i
				}
			}

		}
		ss[8] = strconv.Itoa(minIndex)
		ss[9] = strconv.Itoa(th_results[0])
		ss[10] = strconv.Itoa(th_results[1])
		ss[11] = strconv.Itoa(th_results[2])
		ss[12] = strconv.Itoa(th_results[3])
		ss[13] = strconv.Itoa(th_results[4])
		ss[14] = strconv.Itoa(maxIndex)
		ss[15] = strconv.Itoa(sum)
		ss[16] = avg
		for i := 0; i < len(ss); i++ {
			csvOut.WriteString(ss[i])
			csvOut.WriteString(",")
		}
		csvOut.WriteString("\n")
	}

}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

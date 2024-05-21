package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	start_time_s string
	end_time_s   string
	ftime        string
	exa1         string
	exa2         string
)

func main() {
	ftime = "2022-08-30"
	exa1 = "00:00:00"
	exa2 = "23:59:59"
	msg1 := ftime + " " + exa1
	msg2 := ftime + " " + exa2
	//rd := bufio.NewScanner(os.Stdin)
	//fmt.Println("Please enter the start time(2006-01-02 03:04:05): ")
	//rd.Scan()
	start_time_s = msg1 //rd.Text()
	//fmt.Println("Please enter the end time(2006-01-02 03:04:05): ")
	//rd.Scan()
	end_time_s = msg2 //rd.Text()
	start_time, err := time.Parse("2006-01-02 15:04:05", start_time_s)
	if err != nil {
		panic(err)
	}
	end_time, err := time.Parse("2006-01-02 15:04:05", end_time_s)
	if err != nil {
		panic(err)
	}
	fmt.Println("start time:", start_time.String())
	fmt.Println("end time:", end_time.String())

	start_time_unix := start_time.Unix()
	end_time_unix := end_time.Unix()

	f, err := os.Open("demodata/large1/LDG_N199_Housing TI_fqc_2022-06-20_2022-10-20.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f_csv := csv.NewReader(f)

	m := map[string]bool{}

	f_csv.Read()
	for {
		ss, err := f_csv.Read()
		if err != nil {
			break
		}
		t, err := time.Parse("2006-01-02 15:04:05", ss[1][:19])
		if err != nil {
			panic(err)
		}
		if t.Unix() >= start_time_unix && t.Unix() <= end_time_unix {
			m[ss[0]] = true
		}
	}

	f_dir, err := os.Open("demodata/large1")
	if err != nil {
		panic(err)
	}

	fs, err := f_dir.Readdir(0)
	if err != nil {
		panic(err)
	}

	err = os.Mkdir("demodata/large", 0777)
	if err != nil && errors.Is(err, os.SyscallError{}.Err) {
		log.Println("创建demodata/large目录失败：", err)
		panic(err)
	}

	for i := 0; i < len(fs); i++ {
		fs_r, err := os.Open("demodata/large1/" + fs[i].Name())
		if err != nil {
			log.Println("Open"+fs[i].Name()+"Failed:", err)
			break
		}
		fs_out, err := os.Create("demodata/large/" + fs[i].Name())
		if err != nil {
			log.Println("Create \"demodata/large/\""+fs[i].Name()+"Failed:", err)
			break
		}
		fs_r_csv := csv.NewReader(fs_r)
		ss, err := fs_r_csv.Read()
		if err != nil {
			log.Println("Read Header of"+fs[i].Name()+"Failed:", err)
			continue
		}
		for j := 0; j < len(ss); j++ {
			fs_out.WriteString(ss[j])
			fs_out.WriteString(",")
		}
		fs_out.WriteString("\n")
		for {
			ss, err = fs_r_csv.Read()
			if err != nil {
				break
			}
			if m[ss[0]] {
				for j := 0; j < len(ss); j++ {
					fs_out.WriteString(ss[j])
					fs_out.WriteString(",")
				}
				fs_out.WriteString("\n")
			}
		}
	}

	fmt.Println("已经结束，输入任意键退出")
	//fmt.Scanln()
}

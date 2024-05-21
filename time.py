import schedule


def job1():
    import os

    def fun1():
        os.system("python3 DataClean.py")

    fun1()
def job2():
    import os

    def fun2():
        os.system("go run Time/main.go")

    fun2()

schedule.every().day.at("16:17").do(job1)
schedule.every().day.at("16:20").do(job2)
#schedule.every(5).seconds.do(job)

if __name__ == '__main__':
    while True:
        schedule.run_pending()

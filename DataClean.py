from datetime import datetime
from fileinput import filename
import pandas as pd
import numpy as np
import time
import os
import datetime

    # Configure Folder Name
file_path='./demodata/LF N199 data'
outpath = './demodata/large1'
headers2=['root_serial', '2d-bc-le.local_time', '2d-bc-le.result', '2d-bc-le.insight.test_attributes.uut_start', '2d-bc-le.insight.test_attributes.uut_stop']

file_path=file_path.strip('/')
file_name_list= os.listdir(file_path)
for index,file_name in enumerate(file_name_list):
    if '.csv' not in file_name:
        continue
    headers=[]
    qz=file_name.split('TI_')[-1].split('_')[0]
    df=pd.DataFrame(pd.read_csv(file_path+'/'+file_name, low_memory=False))
    t_list=[]
    for i in df[qz+".local_time"]:
        i=str(i)
        if "/" in i:
            t1=time.strptime(i,"%Y/%m/%d %H:%M:%S")
            t2=time.strftime("%Y-%m-%d %H:%M:%S",t1)
            t_list.append(t2)
            df[qz+".local_time"]=pd.Series(t_list)

    headers3=[x.split('.')[-1] for x in   df.columns.tolist()]
    for h in headers2:
        if h.split('.')[-1] in headers3:
            headers.append(df.columns.tolist()[headers3.index(h.split('.')[-1])])
        else:
            headers.append(h.replace('2d-bc-le',qz))

    for h in headers[-2:]:
        if h not in df.columns.tolist():
            df[h]=df['2d-bc-le.local_time'.replace('2d-bc-le',qz)]
    for h in headers:
        if h not in df.columns.tolist():
            df[h]=''

    df.replace('pass',"passed",inplace=True)
    headers[2] = qz + ".result"
    df[headers].to_csv(outpath+'/'+file_name,index=False)
    print(index+1,len(file_name_list),file_name,'Done')
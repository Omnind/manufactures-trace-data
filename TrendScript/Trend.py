import os, glob
import time
import pandas as pd
import numpy as np

file_path='./Distribution'
outpath = './Trend'

list_a = np.arange(2400+17)

file_path=file_path.strip('/')
file_name_list= os.listdir(file_path)
for index,file_name in enumerate(file_name_list):
    if '.csv' not in file_name:
        continue
    df=pd.read_csv(file_path+'/'+file_name, low_memory=False,usecols=list_a)

    df = df[df['Start Station'] == "CNC Input"]
    df = df[df['End Station'] == "FQC"]
    df=df.drop(['Start Station','End Station', 'Parts','Average'],axis=1)

    df.to_csv(outpath+'/'+file_name,index=False)

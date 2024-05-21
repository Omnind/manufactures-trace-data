import time
import os, glob
import pandas as pd
import numpy as np

path = "./Distribution"
path2 = "./Trend"
outpath = r'./Final Workbook/'

list_a = np.arange(2400+17)
list_b = np.arange(2400+13)

all_files = glob.glob(os.path.join(path, "*.csv"))
all_files2 = glob.glob(os.path.join(path2, "*.csv"))

df_from_each_file = (pd.read_csv(f, sep=',',usecols=list_a) for f in all_files)
df_from_each_file2 = (pd.read_csv(f, sep=',',usecols=list_b) for f in all_files2)

df_merged   = pd.concat(df_from_each_file, ignore_index=True)
df_merged2 = pd.concat(df_from_each_file2, ignore_index=True)

df_merged.to_csv(outpath + "Leadtime Distribution"+".csv",index=False)
df_merged2.to_csv(outpath + "Leadtime Trend"+".csv",index=False)

#df_merged.to_csv(outpath + "AAAA_"+(time.strftime("%Y-%m-%d")+".csv"))
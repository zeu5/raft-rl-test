import re
import sys
import os
import numpy as np
import pandas as pd
import json

def categorize_bugs(directory):
    data = {}
    with open(os.path.join(directory, "0_bug.json")) as bug_file:
        data = json.load(bug_file)
    return data

def collate_occurrences(parent_dir):
    occurs = {}
    for run in os.listdir(parent_dir):
        run_dir = os.path.join(parent_dir, run)
        run_crash_dir = os.path.join(run_dir, "bugs")
        if not os.path.isdir(run_dir) or not os.path.isdir(run_crash_dir):
            continue
        occurs[run] = categorize_bugs(run_crash_dir)
    
    runs = len(list(occurs.values()))
    # occurrences for algo, bug
    data = {}
    for occur in occurs.values():
        for algo, v in occur.items():
            for bug, instances in v.items():
                if (algo, bug) not in data:
                    data[(algo, bug)] = []
                data[(algo,bug)].append(instances)
    
    avg_occurence = {}
    df_data = {
        "Bugs": []
    }
    # First collect all bug names
    for (algo, bug) in data:
        if len(data[(algo, bug)]) < runs:
            data[(algo, bug)] += ([[]]* (runs-len(data[(algo, bug)])))
        avg_occurence[(algo, bug)] = np.mean([len(r) for r in data[(algo, bug)]])
        if bug not in df_data["Bugs"]:
            df_data["Bugs"].append(bug)
        if algo not in df_data:
            df_data[algo] = []

    # Initialize 0's
    for algo in df_data:
        if algo == "Bugs":
            continue
        df_data[algo] = [0]*len(df_data["Bugs"])
    
    # Assign the occurrence 
    for (algo, bug) in avg_occurence:
        bug_index = df_data["Bugs"].index(bug)
        df_data[algo][bug_index] = avg_occurence[(algo, bug)]
        
    df = pd.DataFrame(df_data)
    print(df)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: redis_bugs.py <analysis_dir>")
        exit(1)
    
    root_dir = sys.argv[1]
    if not os.path.isdir(root_dir):
        print("specified path not a directory!")
        exit(1)

    collate_occurrences(sys.argv[1])
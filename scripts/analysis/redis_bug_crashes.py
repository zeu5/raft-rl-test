import re
import sys
import os
import numpy as np
import pandas as pd

def get_stack_trace(log_lines):
    started = False
    stack_traces_lines = []
    cur_stack_trace = []
    for line in log_lines:
        if "Backtrace:" in line:
            started = True
            cur_stack_trace = []
            continue
        if line.strip() == "" and started:
            stack_traces_lines.append([line for line in cur_stack_trace])
            cur_stack_trace = []
            started = False
        if not started:
            continue

        stripped_line = line.strip()
        if stripped_line != "":
            cur_stack_trace.append(stripped_line)
    
    stack_traces = []
    for stack_trace_lines in stack_traces_lines:
        pattern = "(.*)(\(.*\)?)(\[.*\])"
        stack_trace = []
        stack_trace_key = ""
        for line in stack_trace_lines:
            m = re.match(pattern, line)
            if m is None or len(m.group(0)) != len(line):
                continue
            call_file = m.group(1)
            func_call = m.group(2)
            if "redis-server" in call_file:
                call_file = "redis-server"
            if "linux" in call_file:
                continue
            func_call = func_call.strip("()")
            fname = ""
            addr = func_call.strip("()").strip("+")
            if "+" in func_call:
                [fname, addr] = func_call.split("+")
            stack_trace.append("{},{},{}\n".format(call_file, fname, addr))
            stack_trace_key += "{},{};".format(call_file,fname)
        stack_traces.append((hash(stack_trace_key), stack_trace))
        
    return stack_traces

def get_bug_class(trace):
    full_trace = "".join(trace)
    bug_class = "unclassified"
    if ",redis_test_" in full_trace or "base64Encode" in full_trace:
        bug_class = "ignore"
    elif "raft_restore_log" in full_trace or "callRaftPeriodic" in full_trace:
        bug_class = "raft_restore_log"
    elif "raft_node_set_addition_committed" in full_trace:
        bug_class = "raft_apply_entry"
    elif "raft_get_entry_from_idx" in full_trace:
        bug_class = "raft_get_entry_from_idx"
    elif "handleBeforeSleep" in full_trace or "raft_flush" in full_trace:
        bug_class = "handleBeforeSleep"
    elif "raft_node_set_addition_committed" in full_trace:
        bug_class = "raft_node_set_addition_committed"
    elif "raft_node_set_snapshot_offset" in full_trace:
        bug_class = "raft_node_set_snapshot_offset"
    elif "raft_append_entry" in full_trace or "raftNotifyMembershipEvent" in full_trace or "raft_entry_new" in full_trace:
        bug_class = "raft_append_entry"
    elif "ConnIsConnected" in full_trace or "callHandleNodeStates" in full_trace or "HandleNodeStates" in full_trace:
        bug_class = "ConnIsConnected"
    elif "raft_become_follower" in full_trace and "raft_recv_requestvote_response" in full_trace:
        bug_class = "raft_become_follower_requestvote_response"
    elif "raft_become_follower" in full_trace:
        bug_class = "raft_become_follower"
    elif "clientsCron" in full_trace:
        bug_class = "clientsCron"
    elif "raft_set_current_term" in full_trace and "raft_recv_appendentries" in full_trace:
        bug_class = "raft_set_current_term_appendentries"
    elif "raft_set_current_term" in full_trace and "raft_recv_requestvote_response" in full_trace:
        bug_class = "raft_set_current_term_requestvote_response"
    elif "raft_set_current_term" in full_trace and "raft_recv_requestvote" in full_trace:
        bug_class = "raft_set_current_term_requestvote"
    elif "raft_delete_entry_from_idx" in full_trace:
        bug_class = "raft_delete_entry_from_idx"
    elif "redis_test_http_do" in full_trace or "get_message_id" in full_trace or "mg_mgr" in full_trace or "deserializeAEReq" in full_trace:
        bug_class = "ignore"
    elif "libjson-c.so.5" in full_trace or "redis_test_cdeque" in full_trace or "redisraft.so" not in full_trace or "printCrashReport" in full_trace:
        bug_class = "ignore"
    return bug_class

def categorize_bugs(directory):
    print("Analyzing: {}".format(directory))
    bug_files = []
    episode_occurs = {}
    for file in os.listdir(directory):
        parts = file.split("_")
        key = "_".join(parts[:-2])
        if key not in episode_occurs:
            episode_occurs[key] = 0
        step = int(parts[-2][6:])
        if step > episode_occurs[key]:
            episode_occurs[key] = step

    for k,v in episode_occurs.items():
        bug_files.append("{}_epStep{}_bug.log".format(k,v))

    bug_occurrences = {}
    for file in bug_files:
        parts = file.split("_")
        episode = int(parts[-4][2:])
        timeStep = int(parts[-3][2:])
        algo_name = "_".join(parts[1:-4])
        with open(os.path.join(directory, file), encoding = "ISO-8859-1") as log_file:
            traces = get_stack_trace(log_file.readlines())
            for t in traces:
                bug_class = get_bug_class(t[1])
                if bug_class == "unclassified" or bug_class == "ignore":
                    continue
                
                if bug_class not in bug_occurrences:
                    bug_occurrences[bug_class] = {}
                
                if algo_name not in bug_occurrences[bug_class]:
                    bug_occurrences[bug_class][algo_name] = []

                bug_occurrences[bug_class][algo_name].append((episode, timeStep))

    return bug_occurrences 

def print_bug_occurrences(occurs):
    for bug, v in occurs.items():
        print("Occurrences of {}".format(bug))        
        for algo, instances in v.items():
            print("\t{}:{}".format(algo, len(instances)))

def collate_occurrences(parent_dir):
    occurs = {}
    for run in os.listdir(parent_dir):
        run_dir = os.path.join(parent_dir, run)
        run_crash_dir = os.path.join(run_dir, "crash")
        if not os.path.isdir(run_dir) or not os.path.isdir(run_crash_dir):
            continue
        occurs[run] = categorize_bugs(run_crash_dir)
    
    # occurrences for algo, bug
    runs = len(list(occurs.values()))
    data = {}
    for occur in occurs.values():
        for bug, v in occur.items():
            for algo, instances in v.items():
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

    # crashes_dir = os.path.join(root_dir, "crash")
    # if not os.path.exists(crashes_dir):
    #     print("cannot find crashes directory!")
    #     exit(1)
    
    # occurrences = categorize_bugs(crashes_dir)
    # print_bug_occurrences(occurrences)
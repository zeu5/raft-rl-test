import json
import sys

def analyze(graph):
    start_states = set()
    for n in graph["Nodes"].keys():
        node = graph["Nodes"][n]
        if "Prev" not in node or len(node["Prev"].keys()) == 0:
            start_states.add(n)
        prev = set([p for _, v in node["Prev"].items() for p in v ])
        if n in prev:
            prev.remove(n)
        if len(prev) == 0:
            start_states.add(n)
            
    nodes = graph["Nodes"]
    q = list(start_states)
    edges = dict()

    while len(q) != 0:
        cur = q.pop(0)
        depths = set()
        if "Depth" in nodes[cur]:
            continue
        
        if cur in start_states:
            nodes[cur]["Depth"] = 0
        else:
            for _, v in nodes[cur]["Prev"].items():
                for s in v:
                    if "Depth" in nodes[s]:
                        depths.add(nodes[s]["Depth"]+1)
            if len(depths) != 0:
                min_depth = min(list(depths))
                nodes[cur]["Depth"] = min_depth
    
        if "Next" in nodes[cur]:
            for a, v in nodes[cur]["Next"].items():
                for next in v:
                    edges[(cur, a, next)] = [cur,a,next]
                    q.append(next)

    depths = dict()
    for node in nodes.values():
        if "Sibling" in node:
            continue
        if node["Depth"] not in depths:
            depths[node["Depth"]] = set()
        depths[node["Depth"]].add(node["Key"])
    
    for d in depths:
        d_nodes = list(depths[d])
        d_nodes.sort()
        for (i,n) in enumerate(d_nodes):
            nodes[str(n)]["Sibling"] = i

    new_graph = {
        "Nodes": nodes,
        "StartStates": list(start_states),
        "Edges": [edges[e] for e in edges],
    }
    return new_graph

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: python analyze_graph.py <graph_file>")
        sys.exit(1)
    
    file_path = sys.argv[1]
    try:
        graph = json.load(open(file_path))
    except Exception as e:
        print("error reading graph json: ", str(e))
        sys.exit(1)
    
    new_graph = analyze(graph)
    with open(file_path, "w") as f:
        json.dump(new_graph, f)

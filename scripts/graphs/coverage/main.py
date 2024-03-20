import plot as p
import json_manager as jm
import sys
from os import listdir
from os import path

if len(sys.argv) < 2:
    print("Usage: python main.py [old] <folder>")
    sys.exit()
if len(sys.argv) > 3:
    print("Usage: python main.py [old] <folder>")
    sys.exit()
if len(sys.argv) == 3 and sys.argv[1] != "old":
    print("Usage: python main.py [old] <folder>")
    sys.exit()
if len(sys.argv) == 3:
    folder = sys.argv[2]

    config = jm.load_json_file(folder + "/comparison_config.json")

    horizon = config["horizon"]
    nEpisodes = config["episodes"]

    for file in listdir(folder):
        if file.endswith("data.json") and file != "comparison_config.json":
            data = jm.load_json_file(folder + "/" + file)
            plot = p.multilinePlot(data, file.split(".")[0], nEpisodes, horizon)
            plot.savefig(folder + "/" + file.split(".")[0] + "-coveragePlot" + ".png", bbox_inches="tight")
else:
    folder = sys.argv[1]

    config = jm.load_json_file(folder + "/comparison_config.json")

    horizon = config["horizon"]
    nEpisodes = config["episodes"]

    coverageFolder = path.join(folder, "coverage")

    for file in listdir(coverageFolder):
        if file.endswith("data.json") and file != "comparison_config.json":
            data = jm.load_json_file(coverageFolder + "/" + file)
            plot = p.multilinePlotShortest(data, file.split(".")[0], nEpisodes, horizon)
            plot.savefig(coverageFolder + "/" + file.split(".")[0] + "-coveragePlot" + ".png", bbox_inches="tight")
            plot.close()
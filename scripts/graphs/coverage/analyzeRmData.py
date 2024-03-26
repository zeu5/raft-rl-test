import sys
from os import listdir
import os
import json_manager
import plot
import numpy as np

def aggregateStats(dataSet):
    result = {}
    for k, v in dataSet.items():
        if k == "rmStateVisits":
            result["rmStateVisits"] = {}
            result["rmStateVisits"]["FinalAvg"] = np.mean(v["Final"])
            result["rmStateVisits"]["FinalSD"] = np.std(v["Final"])
            result["rmStateVisits"]["InitAvg"] = np.mean(v["Init"])
            result["rmStateVisits"]["InitSD"] = np.std(v["Init"])
            result["rmStateVisits"]["OverallAccuracy"] = result["rmStateVisits"]["FinalAvg"] / result["rmStateVisits"]["InitAvg"]
        else:
            result[k + "Avg"] = np.mean(v)
            result[k + "SD"] = np.std(v)

    return result

# target folder, it should contain all the target .json files
folder = sys.argv[1]
folder = folder + "/rmStatsData"

for expFolder in listdir(folder):
    expFolderPath = folder + "/" + expFolder
    if os.path.isdir(expFolderPath):
        destinationFolder = expFolderPath + "/outputs"
        if not os.path.exists(destinationFolder):
            os.makedirs(destinationFolder)
        
        jsonFiles = [f for f in listdir(expFolderPath) if f.endswith(".json")]

        dataSets = {}

        for file in jsonFiles:
            data = json_manager.load_json_file(expFolderPath + "/" + file)
            for expName, expData in data.items():
                if expName not in dataSets:
                    dataSets[expName] = {}
                    dataSets[expName]["finalPredicateStates"] = []
                    dataSets[expName]["firstEpisodeToFinalPredicate"] = []
                    dataSets[expName]["repeatAccuracy"] = []
                    dataSets[expName]["rmStateVisits"] = {}
                    dataSets[expName]["rmStateVisits"]["Final"] = []
                    dataSets[expName]["rmStateVisits"]["Init"] = []
                for k, v in expData.items():
                    if k == "rmStateVisits":
                        dataSets[expName]["rmStateVisits"]["Final"].append(v.get("Final", 0))
                        dataSets[expName]["rmStateVisits"]["Init"].append(v["Init"])
                    else:
                        dataSets[expName][k].append(v)

        
        json_manager.save_json_file(destinationFolder + "/" + "fullData.json", dataSets)

        aggDataSets = {}
        for k, v in dataSets.items():
            aggDataSets[k] = aggregateStats(v)

        json_manager.save_json_file(destinationFolder + "/" + "aggData.json", aggDataSets)

                




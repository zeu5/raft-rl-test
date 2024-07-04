import sys
from os import listdir
import os
import shutil

def copyToFolder(sourceFile, destFile, targetFolder):
    if not os.path.exists(targetFolder):
        os.makedirs(targetFolder)
    shutil.copyfile(sourceFile, os.path.join(targetFolder,destFile))

# call it on the outer folder of a set of experiments, containing the experiments output folders
# it creates two output folders: genericCoverageData, coverageData and rmStatsData
# it copies the coverage data and the rmStats data from the experiments output folders to the two script output folders
# the .json files are renamed to the experiment name


targetFolder = sys.argv[1]

experimentsFolder = targetFolder # + "/experiments"

# define output folders for the script
destGenCoverageFolder = os.path.join(targetFolder, "genericCoverageData")
if not os.path.exists(destGenCoverageFolder):
    os.makedirs(destGenCoverageFolder)

destCoverageFolder = os.path.join(targetFolder, "coverageData")
if not os.path.exists(destCoverageFolder):
    os.makedirs(destCoverageFolder)

destRmStatsFolder = os.path.join(targetFolder, "rmStatsData")
if not os.path.exists(destRmStatsFolder):
    os.makedirs(destRmStatsFolder)

for file in listdir(experimentsFolder): # foreach experiment folder
    if file == "coverageData" or file == "rmStatsData" or file == "genericCoverageData":
        continue
    if os.path.isdir(os.path.join(experimentsFolder, file)):
        # build the experiment tag
        experimentName = file # name of the experiment output folder defines the name of the experiment

        # specific code for the paper naming format
        # subParts = file.split("_")
        # machineName = subParts[2]
        # experimentName = subParts[3]
        # experimentName = machineName + "_" + experimentName[0] + "_" + experimentName[-1]

    # go to the coverage folder of the experiment
    coverageFolder = os.path.join(experimentsFolder, file, "coverage") # experimentsFolder + "/" + file + "/coverage"
    for coverageFile in listdir(coverageFolder):
        if coverageFile == "0_data.json": # generic coverage
            copyToFolder(os.path.join(coverageFolder, coverageFile), experimentName + ".json", destGenCoverageFolder)
            # copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destCoverageFolder + "/generic")
        # elif coverageFile.endswith("[1]0_abstraction_data.json"):
        #     specificCoverageName = coverageFile.replace("[1]0_abstraction_data.json", "")
        #     copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destCoverageFolder + "/" + specificCoverageName)
        elif coverageFile.endswith("0_abstraction_data.json"):
            specificCoverageName = coverageFile.replace("0_abstraction_data.json", "")
            copyToFolder(os.path.join(coverageFolder, coverageFile), experimentName + ".json", os.path.join(destCoverageFolder, specificCoverageName))
            # copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destCoverageFolder + "/" + specificCoverageName)
        elif coverageFile.endswith("_0.json"):
            specificRmStatsName = coverageFile.replace("_0.json", "")
            copyToFolder(os.path.join(coverageFolder, coverageFile), experimentName + "_rmStats.json", os.path.join(destRmStatsFolder, specificRmStatsName))
            # copyToFolder(coverageFolder + "/" + coverageFile, experimentName + "_rmStats.json", destRmStatsFolder + "/" + specificRmStatsName)
        
    
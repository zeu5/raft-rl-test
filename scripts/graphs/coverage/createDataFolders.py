import sys
from os import listdir
import os
import shutil

def copyToFolder(sourceFile, destFile, targetFolder):
    if not os.path.exists(targetFolder):
        os.makedirs(targetFolder)
    shutil.copyfile(sourceFile, targetFolder + "/" + destFile)

targetFolder = sys.argv[1]

experimentsFolder = targetFolder + "/experiments"

destCoverageFolder = targetFolder + "/coverageData"
if not os.path.exists(destCoverageFolder):
    os.makedirs(destCoverageFolder)

destRmStatsFolder = targetFolder + "/rmStatsData"
if not os.path.exists(destRmStatsFolder):
    os.makedirs(destRmStatsFolder)

for file in listdir(experimentsFolder): # foreach experiment folder
    if os.path.isdir(experimentsFolder + "/" + file):
        # build the experiment tag
        subParts = file.split("_")
        machineName = subParts[2]
        experimentName = subParts[3]
        experimentName = machineName + "_" + experimentName[0] + "_" + experimentName[-1]

    coverageFolder = experimentsFolder + "/" + file + "/coverage"
    for coverageFile in listdir(coverageFolder):
        if coverageFile == "0_data.json": # generic coverage
            copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destCoverageFolder + "/generic")
        elif coverageFile.endswith("[1]0_data.json"):
            specificCoverageName = coverageFile.replace("[1]0_data.json", "")
            copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destCoverageFolder + "/" + specificCoverageName)
        elif coverageFile.endswith("[1]_0.json"):
            specificRmStatsName = coverageFile.replace("[1]_0.json", "")
            copyToFolder(coverageFolder + "/" + coverageFile, experimentName + "_rmStats.json", destRmStatsFolder + "/" + specificRmStatsName)
        
    
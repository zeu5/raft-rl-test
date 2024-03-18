import matplotlib.pyplot as plt

def multilinePlot(data, plotTitle, nEpisodes, horizon):
    plt.figure()
    x = [i * horizon for i in range(nEpisodes)]

    plt.xlabel("Time steps")
    plt.ylabel("Unique states")
    plt.title(plotTitle)

    for expName, entries in data.items():
        if len(entries) > nEpisodes:
            entries = entries[:nEpisodes]
        x_prime = x.copy()
        if len(entries) < len(x):
            x_prime = x[:len(entries)]

        plt.plot(x_prime, entries, label=expName)

    plt.legend(bbox_to_anchor=(1.04, 1), loc="upper left")
    
    return plt
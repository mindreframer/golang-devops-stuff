import csv
import pylab
import numpy as np

records = []

with open("data.csv") as infile:
    reader = csv.DictReader(infile)
    for row in reader:
        row["# Store Nodes"] = int(row["# Store Nodes"])
        row["# Concurrent Requests"] = int(row["# Concurrent Requests"])
        row["Record Size (bytes)"] = int(row["Record Size (bytes)"])
        row["Num Records Generated"] = int(row["Num Records Generated"])
        row["Write Records/s"] = float(row["Write Records/s"])
        row["Sigma Write Records/s"] = float(row["Sigma Write Records/s"])
        row["Write MB/s"] = float(row["Write MB/s"])
        row["Sigma Write MB/s"] = float(row["Sigma Write MB/s"])
        row["Read Records/s"] = float(row["Read Records/s"])
        row["Sigma Read Records/s"] = float(row["Sigma Read Records/s"])
        row["Reads MB/s"] = float(row["Reads MB/s"])
        row["Sigma Reads MB/s"] = float(row["Sigma Reads MB/s"])
        records.append(row)

nodeCounts = [1,3,5,7]
colors = [[0,0,0,1.0],[1.0,0,0,1.0],[0,0.5,0,1.0],[0,0,1.0,1.0]]
dashes = ["-", "--", ":"]
lw = 1

def plot(storeType, category, xitem, yitem, yerritem):
    categorySlices = categories[category]
    for i, nodes in enumerate(nodeCounts):
        for j, categorySlice in enumerate(categorySlices):
            x = []
            y = []
            yerr = []
            for record in records:
                if record["Store Type"] == storeType and record["# Store Nodes"] == nodes and record[category] == categorySlice:
                    x.append(record[xitem])
                    y.append(record[yitem])
                    yerr.append(record[yerritem])
            if len(x) == 0:
                continue
            pylab.fill_between(x,np.array(y) + np.array(yerr), np.array(y) - np.array(yerr), color=colors[i], alpha=0.1, linewidth=0)
            pylab.plot(x, y, color=colors[i], linestyle=dashes[j], label="%s %in %i" %(storeType, nodes, categorySlice), linewidth=lw)

nodeCounts = [1, 5]
categories = {
    "# Concurrent Requests":[1, 30],
}

pylab.figure(figsize=[12,12], dpi=600)
pylab.suptitle("Combined Performance")

ax = pylab.subplot(2, 2, 1)
ax.xaxis.set_label_position('top')
ax.xaxis.tick_top()
lw = 1
plot("ETCD", "# Concurrent Requests", "Record Size (bytes)", "Write Records/s", "Sigma Write Records/s")
lw = 2
plot("Zookeeper", "# Concurrent Requests", "Record Size (bytes)", "Write Records/s", "Sigma Write Records/s")
pylab.xlabel("Record Size (bytes)")
pylab.ylabel("Number of Records Written / s")
pylab.yscale('log')
pylab.xscale('log')
pylab.axis([64, 8192, 10, 3e4])

ax = pylab.subplot(2, 2, 2)
ax.xaxis.set_label_position('top')
ax.xaxis.tick_top()
ax.yaxis.set_label_position('right')
ax.yaxis.tick_right()
pylab.yscale('log')
lw = 1
plot("ETCD", "# Concurrent Requests", "Record Size (bytes)", "Write MB/s", "Sigma Write MB/s")
lw = 2
plot("Zookeeper", "# Concurrent Requests", "Record Size (bytes)", "Write MB/s", "Sigma Write MB/s")
pylab.xlabel("Record Size (bytes)")
pylab.ylabel("MB Written / s")
pylab.yscale('log')
pylab.xscale('log')
pylab.axis([64, 8192, 1e-3, 30])


ax = pylab.subplot(2, 2, 3)
lw = 1
plot("ETCD", "# Concurrent Requests", "Record Size (bytes)", "Read Records/s", "Sigma Read Records/s")
lw = 2
plot("Zookeeper", "# Concurrent Requests", "Record Size (bytes)", "Read Records/s", "Sigma Read Records/s")
pylab.xlabel("Record Size (bytes)")
pylab.ylabel("Number of Records Read / s")
pylab.yscale('log')
pylab.xscale('log')
pylab.axis([64, 8192, 5e3, 1.4e5])

ax = pylab.subplot(2, 2, 4)
ax.yaxis.set_label_position('right')
ax.yaxis.tick_right()
lw = 1
plot("ETCD", "# Concurrent Requests", "Record Size (bytes)", "Reads MB/s", "Sigma Reads MB/s")
lw = 2
plot("Zookeeper", "# Concurrent Requests", "Record Size (bytes)", "Reads MB/s", "Sigma Reads MB/s")
pylab.xlabel("Record Size (bytes)")
pylab.ylabel("MB Read / s")
pylab.yscale('log')
pylab.xscale('log')
pylab.axis([64, 8192, 0.3, 50])

pylab.savefig("combined.pdf")

nodeCounts = [1,3,5,7]

categories = {
    "# Concurrent Requests":[1,10,30],
    "Record Size (bytes)":[128, 512, 2048]
}

for store in ["ETCD", "Zookeeper"]:
    pylab.figure(figsize=[12,12], dpi=600)
    pylab.suptitle(store + " Performance By Record Size")

    ax = pylab.subplot(2, 2, 1)
    ax.xaxis.set_label_position('top')
    ax.xaxis.tick_top()
    plot(store, "# Concurrent Requests", "Record Size (bytes)", "Write Records/s", "Sigma Write Records/s")
    pylab.xlabel("Record Size (bytes)")
    pylab.ylabel("Number of Records Written / s")
    pylab.yscale('log')
    pylab.xscale('log')
    pylab.axis([64, 8192, 10, 3e4])
    pylab.legend(loc = "upper right", fontsize=8, frameon=False)

    ax = pylab.subplot(2, 2, 2)
    ax.xaxis.set_label_position('top')
    ax.xaxis.tick_top()
    ax.yaxis.set_label_position('right')
    ax.yaxis.tick_right()
    pylab.yscale('log')
    plot(store, "# Concurrent Requests", "Record Size (bytes)", "Write MB/s", "Sigma Write MB/s")
    pylab.xlabel("Record Size (bytes)")
    pylab.ylabel("MB Written / s")
    pylab.yscale('log')
    pylab.xscale('log')
    pylab.axis([64, 8192, 1e-3, 30])


    ax = pylab.subplot(2, 2, 3)
    plot(store, "# Concurrent Requests", "Record Size (bytes)", "Read Records/s", "Sigma Read Records/s")
    pylab.xlabel("Record Size (bytes)")
    pylab.ylabel("Number of Records Read / s")
    pylab.yscale('log')
    pylab.xscale('log')
    pylab.axis([64, 8192, 5e3, 1.4e5])

    ax = pylab.subplot(2, 2, 4)
    ax.yaxis.set_label_position('right')
    ax.yaxis.tick_right()
    plot(store, "# Concurrent Requests", "Record Size (bytes)", "Reads MB/s", "Sigma Reads MB/s")
    pylab.xlabel("Record Size (bytes)")
    pylab.ylabel("MB Read / s")
    pylab.yscale('log')
    pylab.xscale('log')
    pylab.axis([64, 8192, 0.3, 50])

    pylab.savefig(store + "_store_performance_by_record_size.pdf")


for store in ["ETCD", "Zookeeper"]:
    pylab.figure(figsize=[12,12], dpi=600)
    pylab.suptitle(store + " Performance By Concurrency")

    ax = pylab.subplot(2, 2, 1)
    ax.xaxis.set_label_position('top')
    ax.xaxis.tick_top()
    plot(store, "Record Size (bytes)", "# Concurrent Requests", "Write Records/s", "Sigma Write Records/s")
    pylab.xlabel("# Concurrent Requests")
    pylab.ylabel("Number of Records Written / s")
    pylab.yscale('log')
    pylab.axis([0,32,10, 3e4])
    pylab.legend(loc = "lower right", fontsize=8, frameon=False)

    ax = pylab.subplot(2, 2, 2)
    ax.xaxis.set_label_position('top')
    ax.xaxis.tick_top()
    ax.yaxis.set_label_position('right')
    ax.yaxis.tick_right()
    pylab.yscale('log')
    plot(store, "Record Size (bytes)", "# Concurrent Requests", "Write MB/s", "Sigma Write MB/s")
    pylab.xlabel("# Concurrent Requests")
    pylab.ylabel("MB Written / s")
    pylab.axis([0,32,1e-3, 30])


    ax = pylab.subplot(2, 2, 3)
    plot(store, "Record Size (bytes)", "# Concurrent Requests", "Read Records/s", "Sigma Read Records/s")
    pylab.xlabel("# Concurrent Requests")
    pylab.ylabel("Number of Records Read / s")
    pylab.yscale('log')
    pylab.axis([0,32,5e3, 1.4e5])

    ax = pylab.subplot(2, 2, 4)
    ax.yaxis.set_label_position('right')
    ax.yaxis.tick_right()
    plot(store, "Record Size (bytes)", "# Concurrent Requests", "Reads MB/s", "Sigma Reads MB/s")
    pylab.xlabel("# Concurrent Requests")
    pylab.ylabel("MB Read / s")
    pylab.yscale('log')
    pylab.axis([0,32,0.3, 50])

    pylab.savefig(store + "_store_performance_by_concurrency.pdf")

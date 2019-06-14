from textgenrnn import textgenrnn
import sys
import os
import psutil

# Set CPU affinity to use all but two threads
p = psutil.Process()
p.cpu_affinity(p.cpu_affinity()[2:])

# Args:
# 1: Training file path
# 2: Epochs count
# 3: Save path
textgen = textgenrnn()
if os.path.isfile(str(sys.argv[3])): 
	textgen = textgenrnn(str(sys.argv[3]))
textgen.train_from_file(str(sys.argv[1]), num_epochs=int(sys.argv[2]))

textgen.save(str(sys.argv[3]))

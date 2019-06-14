from textgenrnn import textgenrnn
import sys

# Args:
# 1: Load path
# 2: Temperature (float)
# 3: Amount of lines to generate
# 4: Amount of characters to generate (MAX)

textgen = textgenrnn(str(sys.argv[1]))

textgen.generate(int(sys.argv[3]), temperature=float(sys.argv[2]), max_gen_length=int(sys.argv[4]))

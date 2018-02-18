import plotly.plotly as py
import plotly.graph_objs as go


def draw_simplex(points):
	x = [x for (x, _, _) in points]
	y = [y for (_, y, _) in points]
	z = [z for (_, _, z) in points]
	print("X: {0}\n Y: {1}\n Z: {2}\n".format(x, y, z))
	trace = go.Heatmap(x=x, y=y, z=z)
	data=[trace]
	py.iplot(data, filename='basic-heatmap')

def read_simplex(filepath):
	"""read_simplex reads values of the simplex from the given
	filepath. Returns the values of the simplex in the format
	x11,x12,...,x1n,z1
	x21,x22,...,x2n,z2
	...
	xn1,xn2,...,x(n+1)n, zn+1
	where xij is the jth coordinate of the ith point and zi
	is the evaulation of the ith point
	"""
	f = open(filepath, 'r')
	points = []
	while f.readline() != '':
		for line in f:
			print(line in set(['', 'Simplex\n']))
			if line in ['', 'Simplex\n']:
				break
		# Read the points of the simplex
		for line in f:
			if line == 'End\n':
				break
			vals = line.split(',')
			nums = [float(val) for val in vals]
			print('nums are {}'.format(nums))
			points.append(nums)

	return points

def main():
	points = read_simplex('simplex.txt')
	print(points)
	draw_simplex(points)

main()

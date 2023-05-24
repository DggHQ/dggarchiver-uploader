#!/usr/bin/env python3

"""Migrates from old sqlite db to new
"""

import sys
import argparse
import sqlite3


def main(arguments):

	parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter)
	parser.add_argument('-c', '--channel', help="Odysee channel name",
                        default='@odysteve', type=str)
	parser.add_argument('infile', help="Input file", type=str)
	parser.add_argument('outfile', help="Output file",
                        type=str)

	args = parser.parse_args(arguments)

	infile = sqlite3.connect(args.infile)
	cursor1 = infile.cursor()
    
	outfile = sqlite3.connect(args.outfile)
	cursor2 = outfile.cursor()
    
	cursor1.execute('SELECT * from uploaded_vods;')
	vods1 = cursor1.fetchall()

	for vod in vods1:
		platform = ''.join([i for i in vod[11].split('-r-')[1] if not i.isdigit()])
		if len(platform) == 0: platform = 'youtube'
		id = vod[0]
		playback_url = ''
		pub_time = vod[1]
		title = vod[2]
		start_time = vod[3]
		end_time = vod[4]
		thumbnail = vod[6]
		thumbnail_path = vod[7]
		path = vod[8]
		duration = vod[9]
		claim = vod[10]
		lbry_name = vod[11]
		lbry_channel = args.channel
		lbry_normalized_name = vod[12]
		lbry_permanent_url = vod[13]
		cursor2.execute('INSERT INTO uploaded_vods (platform, id, playback_url, pub_time, title, start_time, end_time, thumbnail, thumbnail_path, path, duration, claim, lbry_name, lbry_channel, lbry_normalized_name, lbry_permanent_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
		  (platform, id, playback_url, pub_time, title, start_time, end_time, thumbnail, thumbnail_path, path, duration, claim, lbry_name, lbry_channel, lbry_normalized_name, lbry_permanent_url))

	outfile.commit()

if __name__ == '__main__':
	sys.exit(main(sys.argv[1:]))
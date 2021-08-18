#!/bin/sh

  secondsPerBlock=0.5
  startTime="2021-08-16 19:53:03"
  endTime="2021-08-18 16:05:00"
  startHeight=128346718
  startTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${startTime}" +%s`
  endTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${endTime}" +%s`
  endHeight=`echo "${startHeight} + (${endTimeSec} - ${startTimeSec})/${secondsPerBlock}" | bc`

  echo "vapor current height:" ${startTime} ${startHeight}
  echo "vapor end block height:" ${endTime} ${endHeight}

package cfg

import "image/color"

// PointMergeMaxDistance is the maximum allowed distance between points to be merged.
// IMPORTANT: this needs to be configurable. It's being changed here from the assumption
// of mm units to pixel units, as part of the debugging. In the end all of these operations
// should operate on standardized, real-world mm units, because the end goal is to translate
// the input into something the needle cutter can process, and the limiting factor there
// is the needle width (around 0.8 mm) and the accuracy of the machine itself.
// Original value was 0.01 - the 20.0 is being used for testing.
var PointMergeMaxDistance = 20.0

// IMPORTANT: this is being temporarily changed for debugging pixel-based merging. The unit expected here was mm.
//const maxDist = 0.1
var MergePathMaxDistance = 20.0

//
var UndashMaxDist = 4.0
var UndashMinDist = 0.5
var UndashMaxAngle = 45.0

var VectorizeMinLinePixelLength = 5

var BackgroundColor = color.White

var VectorizeMaxRunLength = 20

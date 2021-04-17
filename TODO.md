# TODO List

* Group parts together to easily move them around
    * and so that cutting goes part by part, for easier recovery in case of problems
      compared to alternative of randomly jumping between segments

* Split paths at intersections to improve planning possibilities?
    * may sometimes want to split a path mid-travel to make a nearby disconnected cut
        * prefer corners to edges when doing this? Want to avoid mid-cut discontinuities.

* Optionally add arrow heads to help visualize planning
    * And dotted gray lines to show travel

* Detect and remove black "circle plus line" symbols that appear on some plans


# DONE:

* Auto-rotate
* Set document size to 20"x30", auto-center content
* Simplify paths
    * Join adjacent line segments


# Defer:

* Replace points with curves where possible (Postpone this one - not necessary, and complicates other stages)

* Remove cyan hatching and include remaining cyan lines for bevels
    * But this is dubious in utility; there's a reason FliteTest doesn't score
      these lines in their speed build kits.
    * Hatching is short parallel line segments separated by about 3.333 mm

* Bonus: read text to detect need for additional and mirrored pieces
    (although these aren't hard to do manually, and they still need
    to be placed within the document somewhere)

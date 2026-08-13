/*
Package sfnt implements the SFNT container format interactions.

PDF, web fonts and firmware images are all valid targets for this package.

# Limits

Only glyf outlines can be subset. A CFF or OTTO font gives [ErrNoOutlines],
which lefevre.Font.OutlineFormat reports ahead of the attempt. Nothing is
rewritten beyond the tables above: no cmap subsetting, no name rewriting, and
no reading of the hinting programs that are copied.
*/
package sfnt

package pgengine

// emptyInPredicate is the false predicate rendered for a FilterIn whose list
// is empty or not a list (ruling S041-02: such a filter matches no rows).
const emptyInPredicate = "1 = 0"

// rejectedExplain is what an explain surface returns instead of SQL when its
// input fails [metaengine.ValidateQueryIdents] or the planned-query builder;
// explains have no error return.
const rejectedExplain = "/* rejected: invalid filter or column */"

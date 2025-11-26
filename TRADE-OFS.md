## Cache being pointer map vs value map?
  - If cache is a value map, it will be copied when it is passed to a function, which can be less efficient but safer since the cache is immutable.
  
  - If cache is a pointer map, it will be passed by reference, which can be more efficient but less safe since the cache can be modified by the function.

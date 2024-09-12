# Fiemap
retrieval on sparse file with multiple extents

## Interface
- FieMap(start, length)([]Extent, error)  ：list physical extents by range
- Fallocate(start, length) : allocate extent
- PunchHole(start, length): deallocate extent

## test

test_sparse can used to play with sparse file behaviour. 
```shell
make
```





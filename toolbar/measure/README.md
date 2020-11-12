Measure is a tool for help precise analysis of the performance of the goroutine, and it's designed for thread-safe. 
Just simply call Start() and defer End() on top of the function it will handle all the stack issue and print the tree
struct result when the root function is ended.

#sample usage
func processBlock() {
	measure.Start()
	defer measure.End()

	......
}


#sample result
|--github.com/bytom/vapor/protocol.(*Chain).processBlock: 9.009746ms (100.00)
  |--github.com/bytom/vapor/protocol.(*Chain).saveBlock: 8.086023ms (89.75)
    |--github.com/bytom/vapor/protocol.(*Chain).validateSign: 1.443966ms (17.86)
    |--github.com/bytom/vapor/protocol/validation.ValidateBlock: 195.333µs (2.42)
      |--github.com/bytom/vapor/protocol/validation.ValidateBlockHeader: 26.48µs (13.56)
      |--github.com/bytom/vapor/protocol/validation.ValidateTxs: 88.312µs (45.21)
  |--github.com/bytom/vapor/protocol.(*Chain).connectBlock: 767.073µs (8.51)
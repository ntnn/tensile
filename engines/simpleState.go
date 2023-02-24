package engines

type simpleState struct {
	done map[string]bool
}

func newSimpleState() *simpleState {
	return &simpleState{
		done: map[string]bool{},
	}
}

// func (ss simpleState) isDone(identity ...string) bool {
// 	for _, identity := range identities {
// 		done, ok := ss.done[identity]
// 		if !ok {
// 			return false
// 		}
// 		if !done {
// 			return false
// 		}
// 	}
// 	return true
// }

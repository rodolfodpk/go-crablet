package dcb

import "fmt"

// DebugPrintAppendCondition prints the internal state of an AppendCondition for debugging.
// This should only be used in tests or examples.
func DebugPrintAppendCondition(cond AppendCondition) {
	if cond == nil {
		fmt.Println("[DEBUG] AppendCondition: <nil>")
		return
	}

	// Try to type assert to the internal appendCondition type
	ac, ok := cond.(*appendCondition)
	if !ok {
		fmt.Printf("[DEBUG] Could not type assert to *appendCondition, got %T\n", cond)
		return
	}

	fmt.Printf("[DEBUG] appendCondition: %+v\n", ac)

	// Print FailIfEventsMatch query items
	if ac.FailIfEventsMatch != nil {
		fmt.Printf("[DEBUG] FailIfEventsMatch: %+v\n", ac.FailIfEventsMatch)
		for i, item := range ac.FailIfEventsMatch.Items {
			fmt.Printf("[DEBUG] QueryItem %d: %#v\n", i, item)
		}
	} else {
		fmt.Println("[DEBUG] FailIfEventsMatch: <nil>")
	}

	// Print AfterCursor
	if ac.AfterCursor != nil {
		fmt.Printf("[DEBUG] AfterCursor: TransactionID=%v, Position=%v\n", ac.AfterCursor.TransactionID, ac.AfterCursor.Position)
	} else {
		fmt.Println("[DEBUG] AfterCursor: <nil>")
	}
}

package cmd

import "fmt"

func printDuplicateEntries(values []HieraKeyFullEntry) {
	matches := LookForDuplicateData(values)

	for _, m := range matches {
		fmt.Println(m.Key)
		fmt.Println("locations:")
		for _, l := range m.Locations {
			println("  - " + l)
		}
		fmt.Println("matches:")
		for _, l := range m.Matches {
			str := fmt.Sprintf("%s: %s - %s", l.Key, l.Path1, l.Path2)
			println("  - " + str)
		}
	}
}

func LookForDuplicateData(values []HieraKeyFullEntry) []HieraMatch {
	matchArr := []HieraMatch{}
	arr := []string{}
	for index1, val1 := range values {
		matches := []HieraKeyFullEntry{val1}
		for index2, val2 := range values {
			if val1.Key == val2.Key && index1 != index2 {
				matches = append(matches, val2)
			}
		}

		if len(matches) > 1 {
			if !stringInSlice(val1.Key, arr) {
				a := HieraMatch{
					Key:       val1.Key,
					Locations: []string{},
					Matches:   []HieraMatchEntry{},
				}
				for index3, v := range matches {
					for _, r := range v.Values {
						if !stringInSlice(r.Path, a.Locations) {
							a.Locations = append(a.Locations, r.Path)
						}
					}
					if index3 < len(matches)-1 {
						entries := CompareTwoHieraEntries(matches[index3], matches[index3+1])
						a.Matches = append(a.Matches, entries...)
					}
				}
				matchArr = append(matchArr, a)
				arr = append(arr, val1.Key)
			}

		}
	}
	return matchArr
}

func CompareTwoHieraEntries(var1 HieraKeyFullEntry, var2 HieraKeyFullEntry) []HieraMatchEntry {
	arr := []HieraMatchEntry{}
	for _, value1 := range var1.Values {
		for _, value2 := range var2.Values {
			if value1.Key == value2.Key && value2.Value == value1.Value {
				a := HieraMatchEntry{
					Path1: value1.Path,
					Path2: value2.Path,
					Key:   value1.Key,
				}
				arr = append(arr, a)
			}
		}
	}
	return arr
}

func PrintKeysNotLookedUp(values []HieraKeyFullEntry) {
	arr := []string{}
	for _, key := range values {
		if !key.InLookup && !stringInSlice(key.Key, arr) {
			arr = append(arr, key.Key)
			fmt.Println(key.Key)
		}
	}

}

//func SetInLookup(values *[]HieraKeyFullEntry) {
//	keys := GetAllKeys()
//
//	for _, e := range keys {
//		for index, e2 := range *values {
//			if stringInSlice(e.Key, e2.SubKeys) {
//				(*values)[index].InLookup = true
//			}
//		}
//	}
//}

func getAllValuesInYaml(paths []string) []HieraKeyFullEntry {
	values := []HieraKeyFullEntry{}
	for _, p := range paths {
		val := getHieraKeyValueEntriesForPath(p)
		values = append(values, val...)
	}

	return values
}

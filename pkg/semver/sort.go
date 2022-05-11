package semver

import (
	"github.com/blang/semver"
)

type SemverSortable interface {
	Len() int
	GetSemver(i int) *semver.Version
	GetSequence(i int) int64
	Swap(i, j int)
}

// Modified bubble sort: instead of comparing adjacent elements, compare the elements at the semvers only.
// Input is assumed to be sorted by sequence so non-semver elements are already in correct order.
func SortVersions(versions SemverSortable) {
	endIndex := versions.Len()
	keepSorting := true
	for keepSorting {
		keepSorting = false
		for j := 0; j < endIndex-1; j++ {
			vj := versions.GetSemver(j)
			if vj == nil {
				continue
			}

			isLessThan := false
			for k := j + 1; k < endIndex; k++ {
				vk := versions.GetSemver(k)
				if vk == nil {
					continue
				}

				isLessThan = vj.LT(*vk)
				if vj.EQ(*vk) {
					isLessThan = versions.GetSequence(j) < versions.GetSequence(k)
				}

				if isLessThan {
					break
				}
			}

			if isLessThan {
				versions.Swap(j, j+1)
				keepSorting = true
			}
		}
	}
}

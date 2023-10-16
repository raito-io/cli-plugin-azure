package storage

//go:generate go run github.com/raito-io/enumer -type=ACLPermission
type ACLPermission uint8

const (
	Execute ACLPermission = 1
	Write   ACLPermission = 2
	Read    ACLPermission = 4
)

type ACLPermissionSet uint8

func NewACLPermissionSet(permissions ...ACLPermission) ACLPermissionSet {
	set := ACLPermissionSet(0)
	for _, permission := range permissions {
		set |= ACLPermissionSet(permission)
	}

	return set
}

func (s ACLPermissionSet) Add(permission ACLPermission) ACLPermissionSet {
	return ACLPermissionSet(int8(s) | int8(permission))
}

func (s ACLPermissionSet) Remove(permission ACLPermission) ACLPermissionSet {
	return ACLPermissionSet(int8(s) & ^int8(permission))
}

func (s ACLPermissionSet) Or(set ACLPermissionSet) ACLPermissionSet {
	return ACLPermissionSet(int8(s) | int8(set))
}

func (s ACLPermissionSet) And(set ACLPermissionSet) ACLPermissionSet {
	return ACLPermissionSet(int8(s) & int8(set))
}

func (s ACLPermissionSet) Contains(permission ACLPermission) bool {
	return int8(s)&int8(permission) == int8(permission)
}

func (s ACLPermissionSet) String() string {
	result := []byte("---")
	shortForm := []byte("rwx")

	aclValues := ACLPermissionValues()

	for i, v := range aclValues {
		resultId := len(aclValues) - 1 - i
		if s.Contains(v) {
			result[resultId] = shortForm[resultId]
		}
	}

	return string(result)
}

type ACLPermissionChanges struct {
	Added   ACLPermissionSet
	Removed ACLPermissionSet
}

func (s *ACLPermissionChanges) AddPermissionSet(set ACLPermissionSet) {
	s.Added = s.Added.Or(set)
}

func (s *ACLPermissionChanges) RemovePermissionSet(set ACLPermissionSet) {
	s.Removed = s.Removed.Or(set)
}

func (s *ACLPermissionChanges) Combine(other *ACLPermissionChanges) {
	s.Added = s.Added.Or(other.Added)
	s.Removed = s.Removed.Or(other.Removed)
}

type ACLPermissionChangesWithAP struct {
	ACLPermissionChanges
	APIds []string
}

func (s *ACLPermissionChangesWithAP) AddPermissionSet(set ACLPermissionSet, apId string) {
	s.ACLPermissionChanges.AddPermissionSet(set)
	s.APIds = append(s.APIds, apId)
}

func (s *ACLPermissionChangesWithAP) RemovePermissionSet(set ACLPermissionSet, apId string) {
	s.ACLPermissionChanges.RemovePermissionSet(set)
	s.APIds = append(s.APIds, apId)
}

func (s *ACLPermissionChangesWithAP) Combine(other *ACLPermissionChanges, apId string) {
	s.ACLPermissionChanges.Combine(other)
	s.APIds = append(s.APIds, apId)
}

// ChangeSet Returns the combined permission set. If the second argument is true then the permissions should be removed otherwise the permissions should be added.
func (s *ACLPermissionChanges) ChangeSet() (ACLPermissionSet, bool) {
	if s.Removed > 0 && s.Added == 0 {
		return s.Removed, true
	}

	return s.Added, false
}

type ACLAssignee string

type ACLAssignedItem struct {
	StorageAccount string
	Container      string
	Path           string
}

type ACLAssignment struct {
	Assignee ACLAssignee
	Item     ACLAssignedItem
}

type ACLAssignments map[ACLAssignment]ACLPermissionChanges

func (a ACLAssignments) AddAssignments(assignments ACLAssignments) {
	for assignment := range assignments {
		changes := assignments[assignment]
		if originalChanges, found := a[assignment]; found {
			originalChanges.Combine(&changes)
		} else {
			a[assignment] = changes
		}
	}
}

type ACLAssignmentsWithAP map[ACLAssignment]ACLPermissionChangesWithAP

func (s ACLAssignmentsWithAP) AddAssignments(assignments ACLAssignments, apId string) {
	for assignment := range assignments {
		changes := assignments[assignment]
		if originalChanges, found := s[assignment]; found {
			originalChanges.Combine(&changes, apId)
		} else {
			s[assignment] = ACLPermissionChangesWithAP{changes, []string{apId}}
		}
	}
}

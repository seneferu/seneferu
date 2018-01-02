package memory

import (
	"fmt"

	"gitlab.com/sorenmat/seneferu/model"
)

type MemStorage struct {
	repos []*model.Repo
}

func New() *MemStorage {
	return &MemStorage{}
}

func (m *MemStorage) All() ([]*model.Repo, error) {
	return m.repos, nil
}

func (m *MemStorage) LoadByOrgAndName(org string, name string) (*model.Repo, error) {
	for _, v := range m.repos {
		if v.Name == name && v.Org == org {
			return v, nil
		}
	}
	return nil, fmt.Errorf("unable to find repo")
}
func (m *MemStorage) LoadBuilds(org string, name string) ([]*model.Build, error) {
	return nil, nil
}
func (m *MemStorage) LoadBuild(org string, name string, buildid int) (*model.Build, error) {
	return nil, nil
}
func (m *MemStorage) LoadStep(org string, name string, buildid int, stepname string) (*model.Step, error) {
	return nil, nil
}
func (m *MemStorage) LoadStepInfos(org string, name string, build int) ([]*model.StepInfo, error) {
	return nil, nil
}
func (m *MemStorage) LoadStepInfo(org string, name string, stepname string, build int) (*model.StepInfo, error) {
	return nil, nil
}
func (m *MemStorage) SaveRepo(r *model.Repo) error {
	m.repos = append(m.repos, r)
	return nil
}
func (m *MemStorage) SaveBuild(*model.Build) error {
	return nil
}
func (m *MemStorage) SaveStep(*model.Step) error {
	return nil
}
func (m *MemStorage) GetNextBuildNumber() (int, error) {
	return 1, nil
}
func (m *MemStorage) Close() {

}

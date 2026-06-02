package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TeamAccess struct {
	CurrentUserID  uint
	SharedOwnerIDs []uint
}

func (a TeamAccess) CanModify(ownerID uint) bool {
	return ownerID != 0 && a.CurrentUserID == ownerID
}

func (a TeamAccess) OwnerIDs() []uint {
	ids := make([]uint, 0, len(a.SharedOwnerIDs)+1)
	seen := map[uint]bool{}
	if a.CurrentUserID != 0 {
		ids = append(ids, a.CurrentUserID)
		seen[a.CurrentUserID] = true
	}
	for _, id := range a.SharedOwnerIDs {
		if id == 0 || seen[id] {
			continue
		}
		ids = append(ids, id)
		seen[id] = true
	}
	return ids
}

type TeamService struct {
	db *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{db: db}
}

func (s *TeamService) AccessForUser(userID uint) (TeamAccess, error) {
	var ownerIDs []uint
	err := s.db.Table("team_members").
		Select("DISTINCT teams.owner_id").
		Joins("JOIN teams ON teams.id = team_members.team_id AND teams.deleted_at IS NULL").
		Where("team_members.member_id = ? AND teams.owner_id <> ?", userID, userID).
		Pluck("teams.owner_id", &ownerIDs).Error
	if err != nil {
		return TeamAccess{}, err
	}
	return TeamAccess{CurrentUserID: userID, SharedOwnerIDs: ownerIDs}, nil
}

func (s *TeamService) AccessibleOwnerIDs(userID uint) ([]uint, error) {
	access, err := s.AccessForUser(userID)
	if err != nil {
		return nil, err
	}
	return access.OwnerIDs(), nil
}

func (s *TeamService) EnsureOwnerTeam(ownerID uint) (*model.Team, error) {
	var user model.User
	if err := s.db.First(&user, ownerID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	var team model.Team
	err := s.db.Where("owner_id = ?", ownerID).First(&team).Error
	if err == nil {
		_ = s.ensureMember(team.ID, ownerID)
		return &team, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	team = model.Team{OwnerID: ownerID, Name: user.Username + "的团队"}
	if err := s.db.Create(&team).Error; err != nil {
		return nil, err
	}
	if err := s.ensureMember(team.ID, ownerID); err != nil {
		return nil, err
	}
	return &team, nil
}

func (s *TeamService) MyTeam(ownerID uint) (*model.Team, error) {
	team, err := s.EnsureOwnerTeam(ownerID)
	if err != nil {
		return nil, err
	}
	if err := s.db.Preload("Owner").Preload("Members.Member").First(team, team.ID).Error; err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TeamService) JoinedTeams(memberID uint) ([]model.Team, error) {
	var teams []model.Team
	err := s.db.Model(&model.Team{}).
		Preload("Owner").
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.member_id = ? AND teams.owner_id <> ?", memberID, memberID).
		Order("teams.id DESC").
		Find(&teams).Error
	return teams, err
}

func (s *TeamService) JoinByOwnerUsername(memberID uint, ownerUsername string) (*model.Team, error) {
	ownerUsername = strings.TrimSpace(ownerUsername)
	if ownerUsername == "" {
		return nil, errors.New("请输入团队拥有者用户名")
	}
	var owner model.User
	if err := s.db.Where("username = ? AND status = 'active'", ownerUsername).First(&owner).Error; err != nil {
		return nil, errors.New("团队拥有者不存在或已禁用")
	}
	if owner.ID == memberID {
		return nil, errors.New("不能加入自己的团队")
	}
	team, err := s.EnsureOwnerTeam(owner.ID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureMember(team.ID, memberID); err != nil {
		return nil, err
	}
	if err := s.db.Preload("Owner").First(team, team.ID).Error; err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TeamService) RemoveMember(ownerID, memberID uint) error {
	team, err := s.EnsureOwnerTeam(ownerID)
	if err != nil {
		return err
	}
	if memberID == ownerID {
		return errors.New("不能移除团队拥有者")
	}
	return s.db.Where("team_id = ? AND member_id = ?", team.ID, memberID).Delete(&model.TeamMember{}).Error
}

func (s *TeamService) ensureMember(teamID, memberID uint) error {
	member := model.TeamMember{TeamID: teamID, MemberID: memberID}
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error
}

func accessibleOwnerIDs(db *gorm.DB, currentUserID uint) ([]uint, error) {
	return NewTeamService(db).AccessibleOwnerIDs(currentUserID)
}

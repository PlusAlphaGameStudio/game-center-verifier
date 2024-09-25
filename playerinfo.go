package main

type PlayerInfo struct {
	Kind               string `json:"kind"`
	PlayerId           string `json:"playerId"`
	DisplayName        string `json:"displayName"`
	AvatarImageUrl     string `json:"avatarImageUrl"`
	BannerUrlPortrait  string `json:"bannerUrlPortrait"`
	BannerUrlLandscape string `json:"bannerUrlLandscape"`
	ProfileSettings    struct {
		Kind                  string `json:"kind"`
		ProfileVisible        bool   `json:"profileVisible"`
		FriendsListVisibility string `json:"friendsListVisibility"`
	} `json:"profileSettings"`
	ExperienceInfo struct {
		Kind                       string `json:"kind"`
		CurrentExperiencePoints    string `json:"currentExperiencePoints"`
		LastLevelUpTimestampMillis string `json:"lastLevelUpTimestampMillis"`
		CurrentLevel               struct {
			Kind                string `json:"kind"`
			Level               int    `json:"level"`
			MinExperiencePoints string `json:"minExperiencePoints"`
			MaxExperiencePoints string `json:"maxExperiencePoints"`
		} `json:"currentLevel"`
		NextLevel struct {
			Kind                string `json:"kind"`
			Level               int    `json:"level"`
			MinExperiencePoints string `json:"minExperiencePoints"`
			MaxExperiencePoints string `json:"maxExperiencePoints"`
		} `json:"nextLevel"`
	} `json:"experienceInfo"`
	Title        string `json:"title"`
	GamePlayerId string `json:"gamePlayerId"`
}

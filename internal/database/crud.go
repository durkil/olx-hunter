package database

import "gorm.io/gorm"

func (db *DB) CreateOrUpdateUser(telegramID int64, username, firstName string) (*User, error) {
	user := &User{}

	result := db.Where("telegram_id = ?", telegramID).First(user)

	if result.Error == gorm.ErrRecordNotFound {
		user = &User{
			TelegramID: telegramID,
			Username:   username,
			FirstName:  firstName,
		}
		err := db.Create(user).Error
		return user, err
	}

	user.Username = username
	user.FirstName = firstName
	err := db.Save(user).Error
	return user, err
}

func (db *DB) GetUserByTelegramID(telegramID int64) (*User, error) {
	var user User
	err := db.Where("telegram_id = ?", telegramID).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &user, err
}

func (db *DB) CreateFilter(userID uint, name, query string, minPrice, maxPrice int, city string) (*UserFilter, error) {
	filter := &UserFilter{
		UserID:   userID,
		Name:     name,
		Query:    query,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		City:     city,
		IsActive: true,
	}

	err := db.Create(filter).Error
	return filter, err
}

func (db *DB) GetUserFilters(userID uint) ([]*UserFilter, error) {
	var filters []*UserFilter
	err := db.Where("user_id = ?", userID).Order("created_at desc").Find(&filters).Error
	return filters, err
}

func (db *DB) GetFilterByID(filterID, userID uint) (*UserFilter, error) {
	var filter UserFilter
	err := db.Where("id = ? AND user_id = ?", filterID, userID).First(&filter).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &filter, err
}

func (db *DB) UpdateFilter(filterID, userID uint, name, query string, minPrice, maxPrice int, city string) error {
	return db.Model(&UserFilter{}).
		Where("id = ? AND user_id = ?", filterID, userID).
		Updates(map[string]interface{}{
			"name":      name,
			"query":     query,
			"min_price": minPrice,
			"max_price": maxPrice,
			"city":      city,
		}).Error
}

func (db *DB) DeleteFilter(filterID, userID uint) error {
	return db.Where("id = ? AND user_id = ?", filterID, userID).Delete(&UserFilter{}).Error
}

func (db *DB) ToggleFilter(filterID, userID uint) error {
	return db.Model(&UserFilter{}).
		Where("id = ? AND user_id = ?", filterID, userID).
		Update("is_active", gorm.Expr("NOT is_active")).Error
}

func (db *DB) GetActiveFilters() ([]*UserFilter, error) {
	var filters []*UserFilter
	err := db.Where("is_active = ?", true).Preload("User").Find(&filters).Error
	return filters, err
}


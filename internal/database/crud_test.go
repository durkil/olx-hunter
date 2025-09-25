package database

import (
	"testing"
)

func setupTestDB(t *testing.T) *DB {
    dsn := "host=localhost user=postgres password=password dbname=olx_hunter port=5432 sslmode=disable"
    db, err := Connect(dsn)
    if err != nil {
        t.Fatal("Failed to connect to DB")
    }
    return db
}
func TestCreateOrUpdateUser(t *testing.T) {
	db := setupTestDB(t)

	user, err := db.CreateOrUpdateUser(2306944320, "test", "Test User")
	if err != nil {
		t.Fatal("Error during creating user:", err)
	}

	if user.TelegramID != 2306944320 {
		t.Errorf("Wanted TelegramID=2306944320, got %d", user.TelegramID)
	}

	user2, err := db.CreateOrUpdateUser(2306944320, "updated", "Updated User")
	if err != nil {
		t.Fatal("Error during updating user:", err)
	}

	if user2.FirstName != "Updated User" {
		t.Errorf("Wanted FirstName='Updated User', got '%s'", user2.FirstName)
	}

	if user.ID != user2.ID {
		t.Error("UserID was changed after updating")
	}
	defer func() {
		db.Where("telegram_id = ?", 306944320).Delete(&User{})
	}()
}

func TestGetUserByTelegramID(t *testing.T) {
	db := setupTestDB(t)

	createdUser, err := db.CreateOrUpdateUser(777777, "test2", "Test2 User")
	if err != nil {
		t.Fatal("Error during creating user:", err)
	}

	foundUser, err := db.GetUserByTelegramID(777777)

	if err != nil {
		t.Fatal("Error finding user:", err)
	}

	if foundUser == nil {
		t.Fatal("User does not exist")
	}

	if foundUser.ID != createdUser.ID {
		t.Errorf("Found wrong user. Expected TelegramID=%d, got TelegramID=%d", createdUser.ID, foundUser.ID)
	}

	notFound, err := db.GetUserByTelegramID(92104231235)
	if err != nil {
		t.Fatal("Error shuld be a nil for non-existent user")
	}
	if notFound != nil {
		t.Error("Should return nil, because user does not exist")
	}
	defer func() {
		db.Where("telegram_id = ?", 777777).Delete(&User{})
	}()
}

func TestCreateFilter(t *testing.T) {
	db := setupTestDB(t)
	user, err := db.CreateOrUpdateUser(23512352, "filter", "Filter User")
	if err != nil {
		t.Fatal("Error creating user:", err)
	}

	filter, err := db.CreateFilter(user.ID, "iPhone 15 Test", "iphone-15-pro", 25000, 30000, "Одеса")
	if err != nil {
		t.Fatal("Error creating filter:", err)
	}

	if filter.UserID != user.ID {
		t.Errorf("Expected UserID=%d, got %d", user.ID, filter.UserID)
	}
	if filter.Name != "iPhone 15 Test" {
		t.Errorf("Expected Name='iPhone 15 Test', got '%s'", filter.Name)
	}
	if filter.Query != "iphone-15-pro" {
        t.Errorf("Expected Query='iphone-15', got '%s'", filter.Query)
    }
    if filter.MinPrice != 25000 {
        t.Errorf("Expected MinPrice=25000, got %d", filter.MinPrice)
    }
    if filter.MaxPrice != 30000 {
        t.Errorf("Expected MaxPrice=30000, got %d", filter.MaxPrice)
    }
    if filter.City != "Одеса" {
        t.Errorf("Expected City='Одеса', got '%s'", filter.City)
    }
	if !filter.IsActive {
		t.Error("New filter should be active")
	}
	if filter.ID == 0 {
		t.Error("FilteID should be generated")
	}

	defer func() {
		db.Where("telegram_id = ?", 23512352).Delete(&User{})
	}()
}

func TestGetUserFilters(t *testing.T) {
	db := setupTestDB(t)

	user, err := db.CreateOrUpdateUser(8358453737, "multifilter", "Multi Filter")
	if err != nil {
		t.Fatal("Error creating user:", err)
	}

	filter1, err := db.CreateFilter(user.ID, "Filter 1", "query1", 1000, 2000, "одеса")
	if err != nil {
		t.Fatal("Error creating filter 1:", err)
	}
	filter2, err := db.CreateFilter(user.ID, "Filter 2", "query2", 2000, 3000, "льів")
	if err != nil {
		t.Fatal("Error creating filter 2:", err)
	}

	filters, err := db.GetUserFilters(user.ID)
	if err != nil {
		t.Fatal("Error getting filters:", err)
	}

	if len(filters) < 2 {
		t.Errorf("Expected at least 2 filters, got %d", len(filters))
	}

	foundFilter1 := false
	foundFilter2 := false
	for _, filter := range filters {
		if filter.ID == filter1.ID {
			foundFilter1 = true
		}
		if filter.ID == filter2.ID {
			foundFilter2 = true
		}

		if filter.UserID != user.ID {
			t.Errorf("Filter belongs to wrong user. Expected UserID=%d, got %d", user.ID, filter.UserID)
		}
	}
	if !foundFilter1 || !foundFilter2 {
		t.Error("Not all created filters were found")
	}

	emptyUser, _ := db.CreateOrUpdateUser(54387328, "empty", "Empty User")
	emptyFilters, err := db.GetUserFilters(emptyUser.ID)

	if err != nil {
		t.Fatal("Error getting empty filter:", err)
	}
	if len(emptyFilters) != 0 {
		t.Error("Expected 0 filters for new user")
	}
	
	defer func() {
		db.Where("telegram_id IN ?", []int64{8358453737,54387328}).Delete(&User{})
	}()
}

func TestGetFilterByID(t *testing.T) {
    db := setupTestDB(t)

    user1, _ := db.CreateOrUpdateUser(6666666666, "getfilter", "Get Filter")
    user2, _ := db.CreateOrUpdateUser(7777777777, "other", "Other User")

    filter, err := db.CreateFilter(user1.ID, "Findable Filter", "find-me", 0, 0, "")
    if err != nil {
        t.Fatal("Error creating filter:", err)
    }

    found, err := db.GetFilterByID(filter.ID, user1.ID)
    if err != nil {
        t.Fatal("Error finding filter:", err)
    }
    if found == nil {
        t.Fatal("Filter not found")
    }
    if found.ID != filter.ID {
        t.Errorf("Found wrong filter. Expected ID=%d, got ID=%d", filter.ID, found.ID)
    }

    notFound, err := db.GetFilterByID(filter.ID, user2.ID)
    if err != nil {
        t.Fatal("Error should be nil:", err)
    }
    if notFound != nil {
        t.Error("Should not find other user's filter")
    }

    notExist, err := db.GetFilterByID(99999, user1.ID)
    if err != nil {
        t.Fatal("Error should be nil:", err)
    }
    if notExist != nil {
        t.Error("Should return nil for non-existent filter")
    }

    defer func() {
        db.Where("telegram_id IN ?", []int64{6666666666, 7777777777}).Delete(&User{})
    }()
}

func TestToggleFilter(t *testing.T) {
    db := setupTestDB(t)

    user, _ := db.CreateOrUpdateUser(8888888888, "toggle", "Toggle User")
    filter, err := db.CreateFilter(user.ID, "Toggle Filter", "toggle", 0, 0, "")
    if err != nil {
        t.Fatal("Error creating filter:", err)
    }

    if !filter.IsActive {
        t.Fatal("New filter should be active")
    }

    err = db.ToggleFilter(filter.ID, user.ID)
    if err != nil {
        t.Fatal("Error toggling filter:", err)
    }

    updated, _ := db.GetFilterByID(filter.ID, user.ID)
    if updated.IsActive {
        t.Error("Filter should be inactive after toggle")
    }

    err = db.ToggleFilter(filter.ID, user.ID)
    if err != nil {
        t.Fatal("Error toggling filter back:", err)
    }

    updated2, _ := db.GetFilterByID(filter.ID, user.ID)
    if !updated2.IsActive {
        t.Error("Filter should be active after second toggle")
    }

    defer func() {
        db.Where("telegram_id = ?", 8888888888).Delete(&User{})
    }()
}

func TestGetActiveFilters(t *testing.T) {
    db := setupTestDB(t)

    user1, _ := db.CreateOrUpdateUser(9999999991, "active1", "Active User 1")
    user2, _ := db.CreateOrUpdateUser(9999999992, "active2", "Active User 2")

    activeFilter1, _ := db.CreateFilter(user1.ID, "Active Filter 1", "active1", 0, 0, "")
    activeFilter2, _ := db.CreateFilter(user2.ID, "Active Filter 2", "active2", 0, 0, "")

    inactiveFilter, _ := db.CreateFilter(user1.ID, "Inactive Filter", "inactive", 0, 0, "")
    db.ToggleFilter(inactiveFilter.ID, user1.ID) // Вимкни

    activeFilters, err := db.GetActiveFilters()
    if err != nil {
        t.Fatal("Error getting active filters:", err)
    }

    foundActive1 := false
    foundActive2 := false
    foundInactive := false

    for _, filter := range activeFilters {
        if filter.ID == activeFilter1.ID {
            foundActive1 = true
            if filter.User.ID == 0 {
                t.Error("User should be preloaded for activeFilter1")
            }
            if filter.User.TelegramID != user1.TelegramID {
                t.Error("Wrong user preloaded for activeFilter1")
            }
        }
        if filter.ID == activeFilter2.ID {
            foundActive2 = true
        }
        if filter.ID == inactiveFilter.ID {
            foundInactive = true
        }
    }

    if !foundActive1 {
        t.Error("Active filter 1 should be in active filters")
    }
    if !foundActive2 {
        t.Error("Active filter 2 should be in active filters")
    }
    if foundInactive {
        t.Error("Inactive filter should NOT be in active filters")
    }

    defer func() {
        db.Where("telegram_id IN ?", []int64{9999999991, 9999999992}).Delete(&User{})
    }()
}
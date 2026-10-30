// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package storetest

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/request"
	"github.com/mattermost/mattermost/server/v8/channels/store"
)

func TestRecentSearchStore(t *testing.T, rctx request.CTX, ss store.Store, s SqlStore) {
	t.Run("SaveUpsert", func(t *testing.T) { testRecentSearchSaveUpsert(t, rctx, ss) })
	t.Run("RecencyOrder", func(t *testing.T) { testRecentSearchRecencyOrder(t, rctx, ss) })
	t.Run("Limits", func(t *testing.T) { testRecentSearchLimits(t, rctx, ss) })
	t.Run("TrimTerms", func(t *testing.T) { testRecentSearchTrimTerms(t, rctx, ss) })
	t.Run("OwnerScopedDelete", func(t *testing.T) { testRecentSearchOwnerScopedDelete(t, rctx, ss) })
	t.Run("EmptyTeamId", func(t *testing.T) { testRecentSearchEmptyTeamId(t, rctx, ss) })
	t.Run("LongMultibyteTerms", func(t *testing.T) { testRecentSearchLongMultibyteTerms(t, rctx, ss) })
}

// longRecentSearchTerms returns RecentSearchMaxTermsRunes CJK runes from a fixed seed. Random runes keep
// PostgreSQL from compressing the value below the btree row limit, as a repeated rune would.
func longRecentSearchTerms() string {
	rng := rand.New(rand.NewPCG(2048, 26))
	runes := make([]rune, model.RecentSearchMaxTermsRunes)
	for i := range runes {
		runes[i] = rune(0x4E00 + rng.IntN(0x9FFF-0x4E00))
	}
	return string(runes)
}

func newRecentSearch(userID, terms, searchType, teamID string) *model.RecentSearch {
	rs := &model.RecentSearch{
		UserId:     userID,
		Terms:      terms,
		SearchType: searchType,
		TeamId:     teamID,
	}
	rs.PreSave()
	return rs
}

func testRecentSearchSaveUpsert(t *testing.T, rctx request.CTX, ss store.Store) {
	userID := model.NewId()
	teamID := model.NewId()

	first, err := ss.RecentSearch().Save(newRecentSearch(userID, "hello from:me", model.RecentSearchTypeMessages, teamID))
	require.NoError(t, err)
	require.NotEmpty(t, first.Id)

	time.Sleep(2 * time.Millisecond)

	second, err := ss.RecentSearch().Save(newRecentSearch(userID, "hello from:me", model.RecentSearchTypeMessages, teamID))
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.GreaterOrEqual(t, second.UpdateAt, first.UpdateAt)

	list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, first.Id, list[0].Id)
	require.Equal(t, "hello from:me", list[0].Terms)
}

// testRecentSearchRecencyOrder covers newest-first reads and promotion of a repeated search.
func testRecentSearchRecencyOrder(t *testing.T, rctx request.CTX, ss store.Store) {
	userID := model.NewId()
	teamID := model.NewId()
	var saved []*model.RecentSearch

	// Explicit timestamps avoid ties when several saves occur in the same millisecond.
	for i, terms := range []string{"query-a", "query-b", "query-c"} {
		search := newRecentSearch(userID, terms, model.RecentSearchTypeMessages, teamID)
		search.CreateAt = 1000 + int64(i)
		search.UpdateAt = search.CreateAt
		item, err := ss.RecentSearch().Save(search)
		require.NoError(t, err)
		saved = append(saved, item)
	}

	list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Equal(t, []*model.RecentSearch{saved[2], saved[1], saved[0]}, list)

	// A fresh ID exercises the conflict key used when the same query is submitted again.
	repeated := newRecentSearch(userID, saved[0].Terms, model.RecentSearchTypeMessages, teamID)
	repeated.CreateAt = 2000
	repeated.UpdateAt = 2000
	updated, err := ss.RecentSearch().Save(repeated)
	require.NoError(t, err)
	require.Equal(t, saved[0].Id, updated.Id)
	require.Equal(t, saved[0].CreateAt, updated.CreateAt)
	require.Equal(t, repeated.UpdateAt, updated.UpdateAt)

	// Comparing the entire list checks promotion, uniqueness, and the remaining order together.
	list, err = ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Equal(t, []*model.RecentSearch{updated, saved[2], saved[1]}, list)
}

func testRecentSearchLimits(t *testing.T, rctx request.CTX, ss store.Store) {
	userID := model.NewId()

	list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Empty(t, list)

	// Fixed UpdateAt values make the newest-first order and the pruned row deterministic.
	base := model.GetMillis()
	var saved []*model.RecentSearch
	for i := range model.RecentSearchLimit + 1 {
		rs := newRecentSearch(userID, fmt.Sprintf("query-%d", i), model.RecentSearchTypeMessages, "")
		rs.UpdateAt = base + int64(i)
		item, err := ss.RecentSearch().Save(rs)
		require.NoError(t, err)
		saved = append(saved, item)

		list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
		require.NoError(t, err)
		require.Len(t, list, min(i+1, model.RecentSearchLimit))
		require.Equal(t, item.Id, list[0].Id)
	}

	list, err = ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Len(t, list, model.RecentSearchLimit)
	for i, item := range list {
		require.Equal(t, saved[model.RecentSearchLimit-i].Id, item.Id)
	}
	for _, item := range list {
		require.NotEqual(t, saved[0].Id, item.Id, "oldest search should be pruned")
	}
}

func testRecentSearchTrimTerms(t *testing.T, rctx request.CTX, ss store.Store) {
	userID := model.NewId()
	rs := newRecentSearch(userID, "  spaced terms  ", model.RecentSearchTypeFiles, "")
	require.Equal(t, "spaced terms", rs.Terms)

	saved, err := ss.RecentSearch().Save(rs)
	require.NoError(t, err)
	require.Equal(t, "spaced terms", saved.Terms)

	list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "spaced terms", list[0].Terms)
}

func testRecentSearchOwnerScopedDelete(t *testing.T, rctx request.CTX, ss store.Store) {
	ownerID := model.NewId()
	otherID := model.NewId()

	saved, err := ss.RecentSearch().Save(newRecentSearch(ownerID, "only mine", model.RecentSearchTypeMessages, ""))
	require.NoError(t, err)

	err = ss.RecentSearch().Delete(otherID, saved.Id)
	require.Error(t, err)
	var nf *store.ErrNotFound
	require.ErrorAs(t, err, &nf)

	list, err := ss.RecentSearch().GetByUser(ownerID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Len(t, list, 1)

	err = ss.RecentSearch().Delete(ownerID, saved.Id)
	require.NoError(t, err)

	list, err = ss.RecentSearch().GetByUser(ownerID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Empty(t, list)

	err = ss.RecentSearch().Delete(ownerID, saved.Id)
	require.Error(t, err)
	require.ErrorAs(t, err, &nf)
}

func testRecentSearchEmptyTeamId(t *testing.T, rctx request.CTX, ss store.Store) {
	userID := model.NewId()

	saved, err := ss.RecentSearch().Save(newRecentSearch(userID, "no team", model.RecentSearchTypeMessages, ""))
	require.NoError(t, err)
	require.Equal(t, "", saved.TeamId)

	list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "", list[0].TeamId)
	require.Equal(t, saved.Id, list[0].Id)
}

func testRecentSearchLongMultibyteTerms(t *testing.T, rctx request.CTX, ss store.Store) {
	userID := model.NewId()
	terms := longRecentSearchTerms()
	require.Equal(t, model.RecentSearchMaxTermsRunes, utf8.RuneCountInString(terms))

	first, err := ss.RecentSearch().Save(newRecentSearch(userID, terms, model.RecentSearchTypeMessages, ""))
	require.NoError(t, err)

	second := newRecentSearch(userID, terms, model.RecentSearchTypeMessages, "")
	second.UpdateAt = first.UpdateAt + 1
	saved, err := ss.RecentSearch().Save(second)
	require.NoError(t, err)
	require.Equal(t, first.Id, saved.Id)

	list, err := ss.RecentSearch().GetByUser(userID, model.RecentSearchLimit)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, first.Id, list[0].Id)
	require.Equal(t, terms, list[0].Terms)
}

// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import type {RecentSearch} from '@mattermost/types/search';
import type {Team} from '@mattermost/types/teams';

import {Client4} from 'mattermost-redux/client';

import {
    fireEvent,
    renderWithContext,
    screen,
    userEvent,
    waitFor,
} from 'tests/react_testing_utils';

import RecentSearches from './recent_searches';

const team: Team = {id: 'team1', name: 'team1', display_name: 'Team 1', description: ''} as Team;

function makeItem(overrides: Partial<RecentSearch> = {}): RecentSearch {
    return {
        id: 'id1',
        user_id: 'user1',
        terms: 'hello from:me',
        search_type: 'messages',
        team_id: 'team1',
        create_at: 1,
        update_at: 2,
        ...overrides,
    };
}

describe('components/new_search/RecentSearches', () => {
    const onSearch = jest.fn();

    beforeEach(() => {
        onSearch.mockClear();
    });

    test('hides when empty', () => {
        const {container} = renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
        );
        expect(container).toBeEmptyDOMElement();
    });

    test('renders only five searches in the received order', () => {
        // Deliberately unsorted timestamps ensure the panel preserves the order supplied by Redux.
        const items = [
            makeItem({id: 'a', terms: 'first', update_at: 50}),
            makeItem({id: 'b', terms: 'second', update_at: 40}),
            makeItem({id: 'c', terms: 'third', update_at: 60}),
            makeItem({id: 'd', terms: 'fourth', update_at: 20}),
            makeItem({id: 'e', terms: 'fifth', update_at: 10}),
            makeItem({id: 'f', terms: 'sixth', update_at: 1}),
        ];

        renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
            {entities: {search: {recentSearches: items}}},
        );

        // Accessible rerun buttons expose both the visible query order and the exact display limit.
        const buttons = screen.getAllByRole('button', {name: /^Search again: /});
        expect(buttons.map((button) => button.textContent)).toEqual(['first', 'second', 'third', 'fourth', 'fifth']);
        expect(screen.queryByText('sixth')).not.toBeInTheDocument();
    });

    test('reruns through onSearch and keeps operators', async () => {
        renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
            {entities: {search: {recentSearches: [makeItem()]}}},
        );

        await userEvent.click(screen.getByRole('button', {name: 'Search again: hello from:me'}));
        expect(onSearch).toHaveBeenCalledWith('messages', 'team1', 'hello from:me');
    });

    test('shows an error and does not search when the stored team is missing', async () => {
        renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
            {entities: {search: {recentSearches: [makeItem({team_id: 'gone'})]}}},
        );

        await userEvent.click(screen.getByRole('button', {name: 'Search again: hello from:me'}));
        expect(onSearch).not.toHaveBeenCalled();
        expect(screen.getByText('That team is no longer available')).toBeInTheDocument();
    });

    test('clear does not search', () => {
        renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
            {entities: {search: {recentSearches: [makeItem()]}}},
        );

        // Clear uses pointer-events: none until hover/focus-within; fireEvent bypasses that gate in jsdom.
        fireEvent.click(screen.getByRole('button', {name: 'Clear hello from:me'}));
        expect(onSearch).not.toHaveBeenCalled();
    });

    test('Escape still closes the parent popover', async () => {
        const onClose = jest.fn();
        renderWithContext(
            <div
                onKeyDown={(event) => {
                    if (event.key === 'Escape') {
                        onClose();
                    }
                }}
            >
                <RecentSearches
                    onSearch={onSearch}
                    myTeams={[team]}
                />
            </div>,
            {entities: {search: {recentSearches: [makeItem()]}}},
        );

        screen.getByRole('button', {name: 'Search again: hello from:me'}).focus();
        await userEvent.keyboard('{Escape}');
        expect(onClose).toHaveBeenCalled();
    });

    test('caps the panel height at 550px and scrolls the overflow', () => {
        renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
            {entities: {search: {recentSearches: [makeItem()]}}},
        );

        expect(screen.getByTestId('recentSearches')).toHaveStyle({maxHeight: '550px', overflowY: 'auto'});
    });

    test('reruns an all-teams search with an empty team id and no team warning', async () => {
        renderWithContext(
            <RecentSearches
                onSearch={onSearch}
                myTeams={[team]}
            />,
            {entities: {search: {recentSearches: [makeItem({team_id: '', search_type: 'files', terms: 'report ext:pdf'})]}}},
        );

        await userEvent.click(screen.getByRole('button', {name: 'Search again: report ext:pdf'}));
        expect(onSearch).toHaveBeenCalledWith('files', '', 'report ext:pdf');
        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
        expect(screen.queryByText('That team is no longer available')).not.toBeInTheDocument();
    });

    describe('with the recent searches API mocked', () => {
        const items = [
            makeItem({id: 'a', terms: 'first', update_at: 30}),
            makeItem({id: 'b', terms: 'second', update_at: 20}),
            makeItem({id: 'c', terms: 'third', update_at: 10}),
        ];
        const state = {
            entities: {
                users: {currentUserId: 'user1'},
                search: {recentSearches: items},
            },
        };

        let getRecentSearchesSpy: jest.SpyInstance;
        let deleteRecentSearchSpy: jest.SpyInstance;

        beforeEach(() => {
            // The panel fetches on mount; returning the same list keeps the store deterministic.
            getRecentSearchesSpy = jest.spyOn(Client4, 'getRecentSearches').mockResolvedValue({items});
            deleteRecentSearchSpy = jest.spyOn(Client4, 'deleteRecentSearch').mockResolvedValue({status: 'OK'});
        });

        afterEach(() => {
            // Restore only these spies: restoreAllMocks would also detach the global console spies from setup_jest.
            getRecentSearchesSpy.mockRestore();
            deleteRecentSearchSpy.mockRestore();
        });

        const visibleTerms = () => screen.getAllByRole('button', {name: /^Search again: /}).map((button) => button.textContent);

        test('clear deletes the selected search and keeps the rest in order', async () => {
            renderWithContext(
                <RecentSearches
                    onSearch={onSearch}
                    myTeams={[team]}
                />,
                state,
            );
            await waitFor(() => expect(getRecentSearchesSpy).toHaveBeenCalledTimes(1));

            // Clear uses pointer-events: none until hover/focus-within; fireEvent bypasses that gate in jsdom.
            fireEvent.click(screen.getByRole('button', {name: 'Clear second'}));

            await waitFor(() => expect(screen.queryByText('second')).not.toBeInTheDocument());
            expect(deleteRecentSearchSpy).toHaveBeenCalledTimes(1);
            expect(deleteRecentSearchSpy).toHaveBeenCalledWith('user1', 'b');
            expect(visibleTerms()).toEqual(['first', 'third']);
            expect(onSearch).not.toHaveBeenCalled();
        });

        test('clear is hidden from the pointer by default but reachable and usable from the keyboard', async () => {
            renderWithContext(
                <RecentSearches
                    onSearch={onSearch}
                    myTeams={[team]}
                />,
                state,
            );
            await waitFor(() => expect(getRecentSearchesSpy).toHaveBeenCalledTimes(1));

            const rerun = screen.getByRole('button', {name: 'Search again: first'});
            const clear = screen.getByRole('button', {name: 'Clear first'});

            // This pointer gate is what made the end-to-end test fail without a hover first.
            expect(clear).toHaveStyle({pointerEvents: 'none', opacity: '0'});

            await userEvent.tab();
            expect(rerun).toHaveFocus();
            await userEvent.tab();
            expect(clear).toHaveFocus();

            // jsdom does not apply the :focus-within reveal rule; the Playwright end-to-end test covers it.
            await userEvent.keyboard('{Enter}');

            await waitFor(() => expect(screen.queryByText('first')).not.toBeInTheDocument());
            expect(deleteRecentSearchSpy).toHaveBeenCalledWith('user1', 'a');
            expect(visibleTerms()).toEqual(['second', 'third']);
            expect(onSearch).not.toHaveBeenCalled();
        });
    });
});

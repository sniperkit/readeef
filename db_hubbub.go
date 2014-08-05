package readeef

const (
	get_hubbub_subscription = `
SELECT feed_link, lease_duration, verification_time, subscription_failure
	FROM hubbub_subscriptions WHERE link = $1`
	get_hubbub_subscription_by_feed = `
SELECT link, lease_duration, verification_time, subscription_failure
	FROM hubbub_subscriptions WHERE feed_link = $1`
	create_hubbub_subscription = `
INSERT INTO hubbub_subscriptions(link, feed_link, lease_duration, verification_time, subscription_failure)
	SELECT $1, $2, $3, $4, $5 EXCEPT
	SELECT link, feed_link, lease_duration, verification_time, subscription_failure
		FROM hubbub_subscriptions WHERE link = $1
`
	update_hubbub_subscription = `
UPDATE hubbub_subscriptions SET feed_link = $1, lease_duration = $2,
	verification_time = $3, subscription_failure = $4 WHERE link = $5
`
	delete_hubbub_subscription = `DELETE from hubbub_subscriptions where link = $1`
)

func (db DB) GetHubbubSubscription(link string) (HubbubSubscription, error) {
	var s HubbubSubscription

	if err := db.Get(&s, get_hubbub_subscription, link); err != nil {
		return s, err
	}

	s.Link = link

	return s, nil
}

func (db DB) GetHubbubSubscriptionByFeed(link string) (HubbubSubscription, error) {
	var s HubbubSubscription

	if err := db.Get(&s, get_hubbub_subscription_by_feed, link); err != nil {
		return s, err
	}

	s.FeedLink = link

	return s, nil
}

func (db DB) UpdateHubbubSubscription(s HubbubSubscription) error {
	if err := s.Validate(); err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ustmt, err := tx.Preparex(update_hubbub_subscription)

	if err != nil {
		return err
	}
	defer ustmt.Close()

	_, err = ustmt.Exec(s.FeedLink, s.LeaseDuration, s.VerificationTime, s.SubscriptionFailure, s.Link)
	if err != nil {
		return err
	}

	cstmt, err := tx.Preparex(create_hubbub_subscription)

	if err != nil {
		return err
	}
	defer cstmt.Close()

	_, err = cstmt.Exec(s.Link, s.FeedLink, s.LeaseDuration, s.VerificationTime, s.SubscriptionFailure)
	if err != nil {
		return err
	}

	tx.Commit()

	return nil
}

func (db DB) DeleteHubbubSubscription(s HubbubSubscription) error {
	if err := s.Validate(); err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(delete_hubbub_subscription)

	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.Link)
	if err != nil {
		return err
	}

	tx.Commit()

	return nil
}

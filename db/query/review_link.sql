-- Review links in the post-sale WhatsApp message and the Google data behind
-- the Reviews page (api/review_link.go, api/google_places.go).

-- name: GetReviewSettings :one
SELECT review_place_id, review_base_url, review_title, review_description, google_places_api_key
FROM app_settings WHERE id = 1;

-- name: SetReviewSettings :exec
UPDATE app_settings SET
    review_place_id = sqlc.arg(review_place_id),
    review_base_url = sqlc.arg(review_base_url),
    review_title = sqlc.arg(review_title),
    review_description = sqlc.arg(review_description),
    google_places_api_key = sqlc.arg(google_places_api_key),
    updated_at = now()
WHERE id = 1;

-- name: ListBranchPlaceIDs :many
SELECT id, name, google_place_id FROM branches ORDER BY id;

-- name: SetBranchPlaceID :exec
UPDATE branches SET google_place_id = sqlc.arg(google_place_id) WHERE id = sqlc.arg(id);

-- name: GetWhatsAppMessageByReviewCode :one
SELECT id, branch_id FROM whatsapp_messages WHERE review_code = sqlc.arg(review_code);

-- name: CreateReviewLinkClick :exec
INSERT INTO review_link_clicks (message_id, branch_id, place_id, user_agent)
VALUES (sqlc.narg(message_id), sqlc.narg(branch_id), sqlc.arg(place_id), sqlc.arg(user_agent));

-- name: MarkWhatsAppReviewClicked :exec
UPDATE whatsapp_messages SET review_clicked_at = now()
WHERE id = sqlc.arg(id) AND review_clicked_at IS NULL;

-- name: ReviewFunnel :one
-- Messages sent in the range and how far they got: delivered/read come from
-- Meta's webhook, clicked from the review link.
SELECT
  count(*) FILTER (WHERE status = 'sent')::bigint AS sent,
  count(*) FILTER (WHERE status = 'sent' AND review_code <> '')::bigint AS with_link,
  count(*) FILTER (WHERE delivery_status IN ('delivered', 'read'))::bigint AS delivered,
  count(*) FILTER (WHERE delivery_status = 'read')::bigint AS read,
  count(*) FILTER (WHERE review_clicked_at IS NOT NULL)::bigint AS clicked
FROM whatsapp_messages
WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id));

-- name: ReviewClicksTotal :one
-- Every open of a review link in the range, the plain link included.
SELECT count(*)::bigint
FROM review_link_clicks
WHERE clicked_at >= sqlc.arg(from_at) AND clicked_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id));

-- name: ReviewFunnelByDay :many
-- Per Beirut day: messages sent with a link, and link opens.
WITH sent AS (
  SELECT (created_at AT TIME ZONE 'Asia/Beirut')::date AS day, count(*)::bigint AS n
  FROM whatsapp_messages
  WHERE status = 'sent' AND review_code <> ''
    AND created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
    AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
  GROUP BY 1
), clicks AS (
  SELECT (clicked_at AT TIME ZONE 'Asia/Beirut')::date AS day, count(*)::bigint AS n
  FROM review_link_clicks
  WHERE clicked_at >= sqlc.arg(from_at) AND clicked_at < sqlc.arg(to_at)
    AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
  GROUP BY 1
)
SELECT COALESCE(s.day, c.day)::date AS day,
       COALESCE(s.n, 0)::bigint AS sent,
       COALESCE(c.n, 0)::bigint AS clicks
FROM sent s FULL JOIN clicks c ON c.day = s.day
ORDER BY 1;

-- name: ReviewFunnelByBranch :many
SELECT
  COALESCE(m.branch_id, 0)::bigint AS branch_id,
  COALESCE(b.name, '')::text AS branch_name,
  count(*) FILTER (WHERE m.status = 'sent' AND m.review_code <> '')::bigint AS sent,
  count(*) FILTER (WHERE m.delivery_status = 'read' AND m.review_code <> '')::bigint AS read,
  count(*) FILTER (WHERE m.review_clicked_at IS NOT NULL)::bigint AS clicked
FROM whatsapp_messages m
LEFT JOIN branches b ON b.id = m.branch_id
WHERE m.created_at >= sqlc.arg(from_at) AND m.created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
GROUP BY 1, 2
HAVING count(*) FILTER (WHERE m.status = 'sent' AND m.review_code <> '') > 0
ORDER BY sent DESC, 2;

-- name: LatestGooglePlaceSnapshot :one
SELECT * FROM google_place_snapshots
WHERE place_id = sqlc.arg(place_id)
ORDER BY taken_at DESC, id DESC
LIMIT 1;

-- name: GooglePlaceSnapshotBefore :one
-- The place as it stood at `at`: its last snapshot taken before then.
SELECT * FROM google_place_snapshots
WHERE place_id = sqlc.arg(place_id) AND taken_at < sqlc.arg(at)
ORDER BY taken_at DESC, id DESC
LIMIT 1;

-- name: ListGooglePlaceSnapshots :many
SELECT * FROM google_place_snapshots
WHERE place_id = sqlc.arg(place_id)
  AND taken_at >= sqlc.arg(from_at) AND taken_at < sqlc.arg(to_at)
ORDER BY taken_at, id;

-- name: CreateGooglePlaceSnapshot :exec
INSERT INTO google_place_snapshots (place_id, name, rating, review_count)
VALUES (sqlc.arg(place_id), sqlc.arg(name), sqlc.arg(rating), sqlc.arg(review_count));

-- name: UpsertGoogleReview :exec
INSERT INTO google_reviews (place_id, review_name, author, author_photo, rating, body, published_at)
VALUES (sqlc.arg(place_id), sqlc.arg(review_name), sqlc.arg(author), sqlc.arg(author_photo),
        sqlc.arg(rating), sqlc.arg(body), sqlc.narg(published_at))
ON CONFLICT (place_id, review_name) DO UPDATE SET
    author = EXCLUDED.author,
    author_photo = EXCLUDED.author_photo,
    rating = EXCLUDED.rating,
    body = EXCLUDED.body,
    published_at = EXCLUDED.published_at,
    fetched_at = now();

-- name: ListGoogleReviews :many
-- Newest first, for the given places.
SELECT * FROM google_reviews
WHERE place_id = ANY(sqlc.arg(place_ids)::text[])
ORDER BY published_at DESC NULLS LAST
LIMIT sqlc.arg(row_limit);

-- name: PollRatingTotals :one
-- The WhatsApp rating poll over the range: polls sent, answers, and how many
-- of each star.
SELECT
  count(*) FILTER (WHERE poll_ref <> '')::bigint AS polls,
  count(poll_rating)::bigint AS answers,
  COALESCE(avg(poll_rating), 0)::double precision AS average,
  count(*) FILTER (WHERE poll_rating = 5)::bigint AS stars5,
  count(*) FILTER (WHERE poll_rating = 4)::bigint AS stars4,
  count(*) FILTER (WHERE poll_rating = 3)::bigint AS stars3,
  count(*) FILTER (WHERE poll_rating = 2)::bigint AS stars2,
  count(*) FILTER (WHERE poll_rating = 1)::bigint AS stars1
FROM whatsapp_messages
WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id));

-- name: PollRatingsByBranch :many
SELECT
  COALESCE(m.branch_id, 0)::bigint AS branch_id,
  COALESCE(b.name, '')::text AS branch_name,
  count(m.poll_rating)::bigint AS answers,
  COALESCE(avg(m.poll_rating), 0)::double precision AS average,
  count(*) FILTER (WHERE m.poll_rating = 5)::bigint AS stars5,
  count(*) FILTER (WHERE m.poll_rating = 4)::bigint AS stars4,
  count(*) FILTER (WHERE m.poll_rating = 3)::bigint AS stars3,
  count(*) FILTER (WHERE m.poll_rating = 2)::bigint AS stars2,
  count(*) FILTER (WHERE m.poll_rating = 1)::bigint AS stars1
FROM whatsapp_messages m
LEFT JOIN branches b ON b.id = m.branch_id
WHERE m.created_at >= sqlc.arg(from_at) AND m.created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
  AND m.poll_rating IS NOT NULL
GROUP BY 1, 2
ORDER BY average, 2;

-- name: PollRatingsByDay :many
SELECT
  (poll_voted_at AT TIME ZONE 'Asia/Beirut')::date AS day,
  count(*)::bigint AS answers,
  avg(poll_rating)::double precision AS average,
  count(*) FILTER (WHERE poll_rating = 5)::bigint AS stars5,
  count(*) FILTER (WHERE poll_rating = 4)::bigint AS stars4,
  count(*) FILTER (WHERE poll_rating = 3)::bigint AS stars3,
  count(*) FILTER (WHERE poll_rating = 2)::bigint AS stars2,
  count(*) FILTER (WHERE poll_rating = 1)::bigint AS stars1
FROM whatsapp_messages
WHERE poll_rating IS NOT NULL
  AND created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
GROUP BY 1
ORDER BY 1;

-- name: LowPollRatings :many
-- Clients who answered 1 or 2 stars, newest first: the ones to call back.
SELECT m.id, m.poll_voted_at, m.poll_rating, m.poll_answer, m.client_name, m.client_phone,
       m.invoice_code, m.grand_total_cents, COALESCE(b.name, '')::text AS branch_name,
       -- What they wrote back after the poll, oldest first.
       COALESCE((SELECT string_agg(NULLIF(r.body, ''), E'\n' ORDER BY r.received_at)
                 FROM whatsapp_poll_replies r WHERE r.whatsapp_message_id = m.id), '')::text AS replies
FROM whatsapp_messages m
LEFT JOIN branches b ON b.id = m.branch_id
WHERE m.poll_rating <= 2
  AND m.created_at >= sqlc.arg(from_at) AND m.created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
ORDER BY m.poll_voted_at DESC
LIMIT 20;

-- name: PollReplies :many
-- What clients wrote back after the rating poll, newest first, with their
-- answer: why it wasn't great, and what they looked for and didn't find.
SELECT r.id, r.body, r.kind, r.received_at, m.poll_rating, m.poll_answer,
       m.client_name, m.client_phone, m.invoice_code, COALESCE(b.name, '')::text AS branch_name
FROM whatsapp_poll_replies r
JOIN whatsapp_messages m ON m.id = r.whatsapp_message_id
LEFT JOIN branches b ON b.id = m.branch_id
WHERE r.received_at >= sqlc.arg(from_at) AND r.received_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
ORDER BY r.received_at DESC
LIMIT 100;

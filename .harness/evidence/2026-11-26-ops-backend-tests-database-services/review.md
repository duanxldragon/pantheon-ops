# Review: Backend Tests with MySQL and Redis

Independent CODE/ARCH review: **not performed**. Recorded gap, not an approval.

Check: that the REQUIRED flags are set where a silent skip would otherwise read as green, that the MySQL service is UTC to match `loc=UTC`, that the generic Redis (6379) and the observability Redis (26316) do not collide, and the hosted job duration.

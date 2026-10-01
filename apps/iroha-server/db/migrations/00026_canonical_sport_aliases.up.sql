-- Canonicalize sport aliases to prevent split categories and duplicate buckets:
-- 'running' -> 'run', 'walking' -> 'walk', 'cycling'/'bike' -> 'ride',
-- 'hiking' -> 'hike', 'swimming' -> 'swim', 'FitnessGaming'/'fitnessgaming' -> 'fitness_gaming'.

update tb_activities set sport_type = 'run' where lower(sport_type) in ('running', 'hkworkoutactivitytyperunning');
update tb_activities set sport_type = 'walk' where lower(sport_type) in ('walking', 'hkworkoutactivitytypewalking');
update tb_activities set sport_type = 'ride' where lower(sport_type) in ('cycling', 'bike', 'hkworkoutactivitytypecycling');
update tb_activities set sport_type = 'hike' where lower(sport_type) in ('hiking', 'hkworkoutactivitytypehiking');
update tb_activities set sport_type = 'swim' where lower(sport_type) in ('swimming', 'hkworkoutactivitytypeswimming');
update tb_activities set sport_type = 'fitness_gaming' where lower(sport_type) in ('fitnessgaming', 'fitness gaming', 'hkworkoutactivitytypefitnessgaming');

update tb_activity_observations set sport_type = 'run' where lower(sport_type) in ('running', 'hkworkoutactivitytyperunning');
update tb_activity_observations set sport_type = 'walk' where lower(sport_type) in ('walking', 'hkworkoutactivitytypewalking');
update tb_activity_observations set sport_type = 'ride' where lower(sport_type) in ('cycling', 'bike', 'hkworkoutactivitytypecycling');
update tb_activity_observations set sport_type = 'hike' where lower(sport_type) in ('hiking', 'hkworkoutactivitytypehiking');
update tb_activity_observations set sport_type = 'swim' where lower(sport_type) in ('swimming', 'hkworkoutactivitytypeswimming');
update tb_activity_observations set sport_type = 'fitness_gaming' where lower(sport_type) in ('fitnessgaming', 'fitness gaming', 'hkworkoutactivitytypefitnessgaming');

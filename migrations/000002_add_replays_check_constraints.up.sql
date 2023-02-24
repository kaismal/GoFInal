ALTER TABLE dotareplays.public.replays ADD CONSTRAINT replays_runtime_check CHECK (runtime >= 0);
ALTER TABLE dotareplays.public.replays ADD CONSTRAINT replays_year_check CHECK (year BETWEEN 2011 AND date_part('year', now()));
ALTER TABLE dotareplays.public.replays ADD CONSTRAINT heroes_length_check CHECK (array_length(heroes, 1) BETWEEN 1 AND 11);

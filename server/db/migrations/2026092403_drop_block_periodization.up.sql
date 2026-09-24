-- Block periodization was replaced by plans, drop what it left in settings.
DELETE FROM settings WHERE key LIKE 'blockPeriodization\_%';

-- Migration to add model column to summarizations table

ALTER TABLE summarizations
ADD COLUMN model TEXT DEFAULT '';

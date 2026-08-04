CREATE TABLE task_tags (
    task_pk INT,
    tag_id INT,
    PRIMARY KEY (task_pk, tag_id),
    FOREIGN KEY (task_pk) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

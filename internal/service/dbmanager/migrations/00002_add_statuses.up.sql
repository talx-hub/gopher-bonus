BEGIN TRANSACTION;

      INSERT INTO statuses(name_status)
      VALUES
          ('NEW'),
          ('INVALID'),
          ('PROCESSING'),
          ('PROCESSED');

COMMIT;

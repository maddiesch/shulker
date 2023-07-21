CREATE TABLE "Users" (
  "Username" TEXT NOT NULL UNIQUE,
  "Password" BLOB NOT NULL,
  "Permissions" INTEGER NOT NULL
);

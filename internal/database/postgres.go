package database

func NewConnection(connString string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("Не удалось подключиться к базе данных :%w", err)
	}
	if err = conn.Ping(context.Background()); err !=nil {
		return nil, fmt.Errorf("Не удалось проверить подключение к базе данных: %w", err)
	}

	log.Println("Успешное подключение к базе данных")
	return conn, nil
}
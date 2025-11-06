package doris

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	db     *sql.DB
	config *DorisOutput
	loader Loader
	mu     sync.Mutex
}

func (c *Client) WithLoader(loader Loader) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.loader = loader
	return c
}

type Column struct {
	Name     string
	Type     string
	Nullable bool
}

func (output *DorisOutput) NewClient(ctx context.Context) (*Client, error) {
	var keys []string
	keys = append(keys, output.Host, output.FeHost, output.Username, output.Password, output.Database, output.Table)
	key := strings.Join(keys, ":")
	var (
		db          *sql.DB
		err         error
		dorisClient *Client
	)

	client, ok := pool.Load(key)
	if ok {
		dorisClient = client.(*Client)
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4",
			output.Username,
			output.Password,
			// db client 还是用be node
			// | *************** 但是实际写doris用的是stream load方式，使用的是FeHost ***************** |
			// 这里用host只是用来测试数据库是否能连通，以及表是否存在
			output.Host,
			output.Database,
		)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("connect doris failed: %v", err)
		}

		// 测试连接
		if err = db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping doris failed: %v", err)
		}

		dorisClient = &Client{
			db:     db,
			config: output,
		}

		pool.Store(key, dorisClient)
	}

	return dorisClient, dorisClient.ensureTable(ctx)
}

// ensureTable检查表是否存在,不存在报错
func (c *Client) ensureTable(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查表是否存在
	exists, err := c.tableExists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf(`table "%s.%s" does not exist`, c.config.Database, c.config.Table)
	}

	return nil
}

func (c *Client) tableExists(ctx context.Context) (bool, error) {
	query := `SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = ? AND table_name = ?`

	var count int
	err := c.db.QueryRowContext(ctx, query, c.config.Database, c.config.Table).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

const sortedSetKey = "leaderboard"

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	var gameRank LeaderboardService
	gameRank = NewGameRank(client)

	// 示例调用
	gameRank.UpdateScore("player1", 100, 1743150268969)
	gameRank.UpdateScore("player3", 150, 1743150268969)
	gameRank.UpdateScore("player2", 100, 1743150268970)

	// 获取玩家rank
	fmt.Println(gameRank.GetPlayerRank("player1"))

	// 获取前2名
	fmt.Println(gameRank.GetTopN(2))

	// 获取player1周围的玩家（例如，范围为2）
	fmt.Println(gameRank.GetPlayerRankRange("player1", 2))
}

type RankInfo struct {
	Rank     int
	Score    int64
	PlayerId string
}

type GameRank struct {
	client *redis.Client
}

func NewGameRank(client *redis.Client) *GameRank {
	return &GameRank{client: client}
}

type LeaderboardService interface {
	UpdateScore(playerId string, score int64, timestamp int64)   // 更新玩家分数
	GetPlayerRank(playerId string) RankInfo                      // 获取玩家当前排名
	GetTopN(n int64) []RankInfo                                  // 获取排行榜前N名
	GetPlayerRankRange(playerId string, rangeN int64) []RankInfo // 获取玩家周边排名
}

func (g *GameRank) UpdateScore(playerId string, score int64, timestamp int64) {
	ctx := context.Background()
	newScore := combineScoreAndTimestamp(score, timestamp)
	if _, err := g.client.ZAdd(ctx, sortedSetKey, &redis.Z{
		Score:  newScore,
		Member: playerId,
	}).Result(); err != nil {
		panic(err)
	}
}

func (g *GameRank) GetPlayerRank(playerId string) RankInfo {
	ctx := context.Background()

	rank, err := g.client.ZRevRank(ctx, sortedSetKey, playerId).Result()
	if err == redis.Nil {
		return RankInfo{Rank: -1, PlayerId: playerId} // 玩家不存在
	} else if err != nil {
		panic(err)
	}

	// 获取玩家的分数
	score, err := g.client.ZScore(ctx, sortedSetKey, playerId).Result()
	if err != nil {
		panic(err)
	}

	return RankInfo{
		Rank:     int(rank) + 1, // 转换为 1-based 排名
		Score:    int64(score),
		PlayerId: playerId,
	}
}

func (g *GameRank) GetTopN(n int64) []RankInfo {
	ctx := context.Background()
	results, err := g.client.ZRevRangeWithScores(ctx, sortedSetKey, 0, n-1).Result()
	if err != nil {
		panic(err)
	}

	rankInfos := make([]RankInfo, len(results))
	for i, z := range results {
		rankInfos[i] = RankInfo{
			Rank:     i + 1,
			Score:    int64(z.Score),
			PlayerId: z.Member.(string),
		}
	}
	return rankInfos
}

// 简化后的 GetPlayerRankRange，仅返回玩家周围的排名（示例实现）
func (g *GameRank) GetPlayerRankRange(playerId string, rangeN int64) []RankInfo {
	ctx := context.Background()

	// 获取玩家的排名（降序，0-based）
	rank, err := g.client.ZRevRank(ctx, sortedSetKey, playerId).Result()
	if err != nil {
		return nil
	}

	start := rank - rangeN
	if start < 0 {
		start = 0
	}
	end := rank + rangeN

	// 获取范围内的成员
	members, err := g.client.ZRevRangeWithScores(ctx, sortedSetKey, start, end).Result()
	if err != nil {
		panic(err)
	}

	rankInfos := make([]RankInfo, len(members))
	for i, z := range members {
		rankInfos[i] = RankInfo{
			Rank:     int(start) + i + 1, // 转换为 1-based 排名
			Score:    int64(z.Score),
			PlayerId: z.Member.(string),
		}
	}
	return rankInfos
}

func combineScoreAndTimestamp(score int64, timestamp int64) float64 {
	return float64(score) + (1.0 - float64(timestamp)/1e13)
}

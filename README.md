# gamerank

1.基础功能采用redis zset来实现

2.

3.对于密集排行

（1）zset的score是分数 memeber是用户id的拼接

（2）每次用户更新时先用ZRANGEBYSCORE查出该score是否有member，如果有则删除该member，将该用户id和member拼接成新member再插入

（3）对member操作采用 SETNX保证不会同时对同一个member操作
package com.example.core.repository;

import com.example.core.model.User;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;
import java.util.stream.Collectors;

public class InMemoryUserRepository implements UserRepository {
    
    private final Map<Long, User> users = new ConcurrentHashMap<>();
    private final Map<String, User> usersByEmail = new ConcurrentHashMap<>();
    private final AtomicLong idGenerator = new AtomicLong(1);
    
    @Override
    public User save(User user) {
        if (user.getId() == null) {
            user.setId(idGenerator.getAndIncrement());
        }
        
        users.put(user.getId(), user);
        if (user.getEmail() != null) {
            usersByEmail.put(user.getEmail().toLowerCase(), user);
        }
        
        return user;
    }
    
    @Override
    public Optional<User> findById(Long id) {
        return Optional.ofNullable(users.get(id));
    }
    
    @Override
    public Optional<User> findByEmail(String email) {
        return Optional.ofNullable(usersByEmail.get(email.toLowerCase()));
    }
    
    @Override
    public List<User> findByStatus(User.UserStatus status) {
        return users.values().stream()
                .filter(user -> user.getStatus() == status)
                .collect(Collectors.toList());
    }
    
    @Override
    public List<User> findAll() {
        return new ArrayList<>(users.values());
    }
    
    @Override
    public void deleteById(Long id) {
        User user = users.remove(id);
        if (user != null && user.getEmail() != null) {
            usersByEmail.remove(user.getEmail().toLowerCase());
        }
    }
    
    @Override
    public long count() {
        return users.size();
    }
    
    @Override
    public boolean existsById(Long id) {
        return users.containsKey(id);
    }
    
    @Override
    public boolean existsByEmail(String email) {
        return usersByEmail.containsKey(email.toLowerCase());
    }
}
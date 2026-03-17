package com.example.core.repository;

import com.example.core.model.User;
import java.util.List;
import java.util.Optional;

public interface UserRepository {
    
    User save(User user);
    
    Optional<User> findById(Long id);
    
    Optional<User> findByEmail(String email);
    
    List<User> findByStatus(User.UserStatus status);
    
    List<User> findAll();
    
    void deleteById(Long id);
    
    long count();
    
    boolean existsById(Long id);
    
    boolean existsByEmail(String email);
}